package stream

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	"github.com/quic-s/quics-protocol/pkg/types/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
	"google.golang.org/protobuf/proto"
)

type Stream struct {
	logLevel int
	Stream   quic.Stream
}

func New(logLevel int, stream quic.Stream) (*Stream, error) {
	if stream == nil {
		return nil, fmt.Errorf("stream is nil")
	}
	return &Stream{
		logLevel: logLevel,
		Stream:   stream,
	}, nil
}

func (s *Stream) Close() error {
	err := s.Stream.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *Stream) SendBMessage(data []byte) error {
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}
	err = WriteHeader(s, pb.RequestType_BMESSAGE, requestId)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}

	err = WriteMessage(s, data)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}
	return nil
}

func (s *Stream) SendFile(filePath string) error {
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}
	err = WriteHeader(s, pb.RequestType_FILE, requestId)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}

	err = WriteFile(s, filePath)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}

	err = s.Stream.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *Stream) SendFileBMessage(data []byte, filePath string) error {
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}
	err = WriteHeader(s, pb.RequestType_FILE_BMESSAGE, requestId)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}

	err = WriteMessage(s, data)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}

	err = WriteFile(s, filePath)
	if err != nil {
		s.Stream.CancelWrite(0)
		return err
	}
	return nil
}

func (s *Stream) RecvBMessage() ([]byte, error) {
	header, err := ReadHeader(s)
	if err != nil {
		return nil, err
	}
	if header.RequestType != pb.RequestType_BMESSAGE {
		return nil, fmt.Errorf("request type is not BMessage")
	}

	message, err := ReadMessage(s)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *Stream) RecvFile() (*fileinfo.FileInfo, io.Reader, error) {
	header, err := ReadHeader(s)
	if err != nil {
		return nil, nil, err
	}
	if header.RequestType != pb.RequestType_FILE {
		return nil, nil, fmt.Errorf("request type is not File")
	}

	fileInfo, fileReader, err := ReadFile(s)
	if err != nil {
		return nil, nil, err
	}

	return fileInfo, fileReader, nil
}

func (s *Stream) RecvFileBMessage() ([]byte, *fileinfo.FileInfo, io.Reader, error) {
	header, err := ReadHeader(s)
	if err != nil {
		return nil, nil, nil, err
	}
	if header.RequestType != pb.RequestType_FILE_BMESSAGE {
		return nil, nil, nil, fmt.Errorf("request type is not FileBMessage")
	}

	message, err := ReadMessage(s)
	if err != nil {
		return nil, nil, nil, err
	}

	fileInfo, fileReader, err := ReadFile(s)
	if err != nil {
		return nil, nil, nil, err
	}

	return message, fileInfo, fileReader, nil
}

func WriteHeader(s *Stream, requestType pb.RequestType, requestId []byte) error {
	header := &pb.Header{
		RequestType: requestType,
		RequestId:   requestId,
	}
	headerOut, err := proto.Marshal(header)
	if err != nil {
		return err
	}

	buf := make([]byte, 2, 2+len(headerOut))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(headerOut)))
	buf = append(buf, headerOut...)

	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sending ", cap(buf), "bytes")
	}

	n, err := s.Stream.Write(buf)
	if err != nil {
		return err
	}
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n)
	}
	return nil
}

func WriteMessage(s *Stream, data []byte) error {
	buf := make([]byte, 4, 4+len(data))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(data)))

	buf = append(buf, data...)

	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sending ", cap(buf), "bytes")
	}

	n, err := s.Stream.Write(buf)
	if err != nil {
		return err
	}
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n)
	}
	return nil
}

func WriteFile(s *Stream, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}
	defer file.Close()

	osFileInfo, err := file.Stat()
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}

	qpFileInfo, err := fileinfo.NewFromOSFileInfo(osFileInfo)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}

	pbFileInfo, err := qpFileInfo.ToProtobuf()
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}

	fileInfoOut, err := proto.Marshal(pbFileInfo)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}

	buf := make([]byte, 2, 2+len(fileInfoOut))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(fileInfoOut)))
	buf = append(buf, fileInfoOut...)

	n, err := s.Stream.Write(buf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("write size is not equal to buf size")
	}
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n, "bytes")
	}

	if !pbFileInfo.IsDir {
		fileBuf := bufio.NewReader(file)
		num, err := io.Copy(s.Stream, fileBuf)
		if err != nil {
			log.Println("quics-protocol: ", err)
			return err
		}
		if num != osFileInfo.Size() {
			return fmt.Errorf("write size is not equal to file size")
		}
		if s.logLevel <= qpLog.INFO {
			log.Println("quics-protocol: ", "sent", num, "bytes")
		}
	}
	return nil
}

func WriteTransaction(s *Stream, transactionName string, transactionID []byte) error {
	transaction := &pb.Transaction{
		TransactionName: transactionName,
		TransactionID:   transactionID,
	}
	transactionOut, err := proto.Marshal(transaction)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}

	buf := make([]byte, 2, 2+len(transactionOut))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(transactionOut)))

	buf = append(buf, transactionOut...)

	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sending ", cap(buf), "bytes")
	}

	n, err := s.Stream.Write(buf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return err
	}
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n)
	}
	return nil
}

func ReadHeader(s *Stream) (*pb.Header, error) {
	headerSizeBuf := make([]byte, 2)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read header size")
	}
	n, err := io.ReadFull(s.Stream, headerSizeBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("header size is not 2 bytes")
	}
	headerSize := uint16(binary.BigEndian.Uint16(headerSizeBuf))
	headerBuf := make([]byte, headerSize)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read header")
	}
	n, err = io.ReadFull(s.Stream, headerBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != int(headerSize) {
		return nil, fmt.Errorf("header size is not %d bytes", headerSize)
	}
	header := &pb.Header{}
	proto.Unmarshal(headerBuf, header)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", header.RequestType, header.RequestType, header.RequestId)
	}

	return header, nil
}

func ReadMessage(s *Stream) ([]byte, error) {
	messageSizeBuf := make([]byte, 4)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read message size")
	}
	n, err := io.ReadFull(s.Stream, messageSizeBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != 4 {
		return nil, fmt.Errorf("message size is not 4 bytes")
	}

	messageSize := uint32(binary.BigEndian.Uint32(messageSizeBuf))
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read message")
	}
	messageBuf := make([]byte, messageSize)
	n, err = io.ReadFull(s.Stream, messageBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != int(messageSize) {
		return nil, fmt.Errorf("message size is not %d bytes", messageSize)
	}
	return messageBuf, nil
}

func ReadFile(s *Stream) (*fileinfo.FileInfo, io.Reader, error) {
	fileInfoSizeBuf := make([]byte, 2)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read file info size")
	}
	n, err := io.ReadFull(s.Stream, fileInfoSizeBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, nil, err
	}
	if n != 2 {
		return nil, nil, fmt.Errorf("file info size is not 2 bytes")
	}
	fileInfoSize := uint16(binary.BigEndian.Uint16(fileInfoSizeBuf))

	fileInfoBuf := make([]byte, fileInfoSize)
	n, err = io.ReadFull(s.Stream, fileInfoBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, nil, err
	}
	if n != int(fileInfoSize) {
		return nil, nil, fmt.Errorf("file info size is not %d bytes", fileInfoSize)
	}

	protoFileInfo := &pb.FileInfo{}
	proto.Unmarshal(fileInfoBuf, protoFileInfo)

	fileInfo, err := fileinfo.NewFromProtobuf(protoFileInfo)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, nil, err
	}
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", fileInfo.Name, fileInfo.Size, "bytes")
		log.Println("quics-protocol: ", "read file")
	}

	fileReader := io.LimitReader(s.Stream, fileInfo.Size)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "init file reader with size", fileInfo.Size)
	}

	return fileInfo, fileReader, nil
}

func ReadTransaction(s *Stream) (*pb.Transaction, error) {
	transactionSizeBuf := make([]byte, 2)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read transaction size")
	}
	n, err := io.ReadFull(s.Stream, transactionSizeBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("transaction size is not 2 bytes")
	}
	transactionSize := uint16(binary.BigEndian.Uint16(transactionSizeBuf))

	transactionBuf := make([]byte, transactionSize)
	n, err = io.ReadFull(s.Stream, transactionBuf)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if n != int(transactionSize) {
		return nil, fmt.Errorf("transaction size is not %d bytes", transactionSize)
	}

	transaction := &pb.Transaction{}
	proto.Unmarshal(transactionBuf, transaction)
	if s.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", transaction.TransactionName, transaction.TransactionID)
	}
	return transaction, nil
}
