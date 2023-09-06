package connection

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	"github.com/quic-s/quics-protocol/pkg/utils/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
	"google.golang.org/protobuf/proto"
)

type Connection struct {
	logLevel            int
	writeMut            *sync.Mutex
	MsgResponseChan     map[string]chan []byte
	FileResponseChan    map[string]chan []byte
	FileMsgResponseChan map[string]chan []byte
	Conn                quic.Connection
	Stream              quic.Stream
}

func New(logLevel int, conn quic.Connection, stream quic.Stream) (*Connection, error) {
	if conn == nil {
		return nil, fmt.Errorf("conn is nil")
	}
	if !conn.ConnectionState().TLS.HandshakeComplete {
		return nil, fmt.Errorf("TLS handshake is not completed")
	}
	if stream == nil {
		return nil, fmt.Errorf("stream is nil")
	}

	return &Connection{
		logLevel:            logLevel,
		writeMut:            &sync.Mutex{},
		MsgResponseChan:     make(map[string]chan []byte),
		FileResponseChan:    make(map[string]chan []byte),
		FileMsgResponseChan: make(map[string]chan []byte),
		Conn:                conn,
		Stream:              stream,
	}, nil
}

func (c *Connection) Close() error {
	err := c.Stream.Close()
	if err != nil {
		return err
	}
	err = c.Conn.CloseWithError(0, "Connection closed by peer")
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) CloseWithError(message string) error {
	err := c.Conn.CloseWithError(0, message)
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) ReadHeader() (*pb.Header, error) {
	headerSizeBuf := make([]byte, 2)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read header size")
	}
	n, err := io.ReadFull(c.Stream, headerSizeBuf)
	if err != nil {
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("header size is not 2 bytes")
	}
	headerSize := uint16(binary.BigEndian.Uint16(headerSizeBuf))
	headerBuf := make([]byte, headerSize)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read header")
	}
	n, err = io.ReadFull(c.Stream, headerBuf)
	if err != nil {
		return nil, err
	}
	if n != int(headerSize) {
		return nil, fmt.Errorf("header size is not %d bytes", headerSize)
	}
	header := &pb.Header{}
	proto.Unmarshal(headerBuf, header)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", header.MessageType, header.RequestType, header.RequestId)
	}

	return header, nil
}

func (c *Connection) ReadMessage() (*pb.Message, error) {
	messageSizeBuf := make([]byte, 4)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read message size")
	}
	n, err := io.ReadFull(c.Stream, messageSizeBuf)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, fmt.Errorf("message size is not 4 bytes")
	}

	messageSize := uint32(binary.BigEndian.Uint32(messageSizeBuf))
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read message")
	}
	messageBuf := make([]byte, messageSize)
	n, err = io.ReadFull(c.Stream, messageBuf)
	if err != nil {
		return nil, err
	}
	if n != int(messageSize) {
		return nil, fmt.Errorf("message size is not %d bytes", messageSize)
	}
	message := &pb.Message{}
	proto.Unmarshal(messageBuf, message)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "message data: ", string(message.Data))
	}
	return message, nil
}

func (c *Connection) ReadFile() (*fileinfo.FileInfo, io.Reader, error) {
	fileInfoSizeBuf := make([]byte, 2)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read file info size")
	}
	n, err := io.ReadFull(c.Stream, fileInfoSizeBuf)
	if err != nil {
		return nil, nil, err
	}
	if n != 2 {
		return nil, nil, fmt.Errorf("file info size is not 2 bytes")
	}
	fileInfoSize := uint16(binary.BigEndian.Uint16(fileInfoSizeBuf))

	fileInfoBuf := make([]byte, fileInfoSize)
	n, err = io.ReadFull(c.Stream, fileInfoBuf)
	if err != nil {
		return nil, nil, err
	}
	if n != int(fileInfoSize) {
		return nil, nil, fmt.Errorf("file info size is not %d bytes", fileInfoSize)
	}

	protoFileInfo := &pb.FileInfo{}
	proto.Unmarshal(fileInfoBuf, protoFileInfo)

	fileInfo, err := fileinfo.DecodeFileInfo(protoFileInfo.FileInfo)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", fileInfo.Name, fileInfo.Size, "bytes")
		log.Println("quics-protocol: ", "read file")
	}

	fileReader := io.LimitReader(c.Stream, fileInfo.Size)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "init file reader with size", fileInfo.Size)
	}

	return fileInfo, fileReader, nil
}

func (c *Connection) SendMessage(msgType string, data []byte) error {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return err
	}
	err = c.writeHeader(pb.MessageType_MESSAGE, requestId, msgType)
	if err != nil {
		return err
	}

	err = c.writeMessage(data)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) SendFile(fileType string, filePath string) error {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return err
	}
	err = c.writeHeader(pb.MessageType_FILE, requestId, fileType)
	if err != nil {
		return err
	}

	err = c.writeFile(filePath)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) SendFileMessage(fileMsgType string, data []byte, filePath string) error {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return err
	}
	err = c.writeHeader(pb.MessageType_FILE_MESSAGE, requestId, fileMsgType)
	if err != nil {
		return err
	}

	err = c.writeMessage(data)
	if err != nil {
		return err
	}

	err = c.writeFile(filePath)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) SendMessageWithResponse(msgType string, data []byte) ([]byte, error) {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		c.Stream.CancelWrite(0)
		c.writeMut.Unlock()
		return nil, err
	}
	err = c.writeHeader(pb.MessageType_MESSAGE_W_RESPONSE, requestId, msgType)
	if err != nil {
		c.Stream.CancelWrite(0)
		c.writeMut.Unlock()
		return nil, err
	}

	err = c.writeMessage(data)
	if err != nil {
		c.Stream.CancelWrite(0)
		c.writeMut.Unlock()
		return nil, err
	}
	c.writeMut.Unlock()

	requestUUID, err := uuid.FromBytes(requestId)
	if err != nil {
		return nil, err
	}
	matchedChan := make(chan []byte)
	c.MsgResponseChan[requestUUID.String()] = matchedChan
	response := <-c.MsgResponseChan[requestUUID.String()]
	log.Println("quics-protocol: ", "response received")
	delete(c.MsgResponseChan, requestUUID.String())
	return response, nil
}

func (c *Connection) SendFileWithResponse(fileType string, filePath string) ([]byte, error) {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return nil, err
	}
	err = c.writeHeader(pb.MessageType_FILE_W_RESPONSE, requestId, fileType)
	if err != nil {
		return nil, err
	}

	err = c.writeFile(filePath)
	if err != nil {
		return nil, err
	}
	c.writeMut.Unlock()

	requestUUID, err := uuid.FromBytes(requestId)
	if err != nil {
		return nil, err
	}
	matchedChan := make(chan []byte)
	c.FileResponseChan[requestUUID.String()] = matchedChan
	response := <-c.FileResponseChan[requestUUID.String()]
	delete(c.FileResponseChan, requestUUID.String())
	return response, nil
}

func (c *Connection) SendFileMessageWithResponse(fileMsgType string, data []byte, filePath string) ([]byte, error) {
	c.writeMut.Lock()
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return nil, err
	}
	err = c.writeHeader(pb.MessageType_FILE_MESSAGE_W_RESPONSE, requestId, fileMsgType)
	if err != nil {
		return nil, err
	}

	err = c.writeMessage(data)
	if err != nil {
		return nil, err
	}

	err = c.writeFile(filePath)
	if err != nil {
		return nil, err
	}
	c.writeMut.Unlock()

	requestUUID, err := uuid.FromBytes(requestId)
	if err != nil {
		return nil, err
	}
	matchedChan := make(chan []byte)
	c.FileMsgResponseChan[requestUUID.String()] = matchedChan
	response := <-c.FileMsgResponseChan[requestUUID.String()]
	delete(c.FileMsgResponseChan, requestUUID.String())
	return response, nil
}

func (c *Connection) SendResponse(responseType pb.MessageType, requestId []byte, requestType string, data []byte) error {
	c.writeMut.Lock()
	err := c.writeHeader(responseType, requestId, requestType)
	if err != nil {
		return err
	}

	err = c.writeMessage(data)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) writeHeader(messageType pb.MessageType, requestId []byte, requestType string) error {
	header := &pb.Header{
		MessageType: messageType,
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

	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sending ", cap(buf), "bytes")
	}

	n, err := c.Stream.Write(buf)
	if err != nil {
		return err
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n)
	}
	return nil
}

func (c *Connection) writeMessage(data []byte) error {
	message := &pb.Message{
		Data: data,
	}
	messageOut, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	buf := make([]byte, 4, 4+len(messageOut))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(messageOut)))

	buf = append(buf, messageOut...)

	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sending ", cap(buf), "bytes")
	}

	n, err := c.Stream.Write(buf)
	if err != nil {
		return err
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n)
	}
	return nil
}

func (c *Connection) writeFile(filePath string) error {
	osFileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			qpFileInfo := &fileinfo.FileInfo{
				Name: filePath,
				Size: 0,
			}

			bFileInfo, err := fileinfo.EncodeFileInfo(qpFileInfo)
			if err != nil {
				return err
			}

			fileInfo := &pb.FileInfo{
				FileInfo: bFileInfo,
			}

			fileInfoOut, err := proto.Marshal(fileInfo)
			if err != nil {
				return err
			}

			buf := make([]byte, 2, 2+len(fileInfoOut))
			binary.BigEndian.PutUint16(buf[:2], uint16(len(fileInfoOut)))
			buf = append(buf, fileInfoOut...)

			n, err := c.Stream.Write(buf)
			if err != nil {
				return err
			}
			if n != len(buf) {
				return fmt.Errorf("write size is not equal to buf size")
			}
			if c.logLevel <= qpLog.INFO {
				log.Println("quics-protocol: ", "sent", n, "bytes")
			}

			return nil
		}
		return err
	}
	bFileInfo, err := fileinfo.EncodeFromOsFileInfo(osFileInfo)
	if err != nil {
		return err
	}

	fileInfo := &pb.FileInfo{
		FileInfo: bFileInfo,
	}

	fileInfoOut, err := proto.Marshal(fileInfo)
	if err != nil {
		return err
	}

	buf := make([]byte, 2, 2+len(fileInfoOut))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(fileInfoOut)))
	buf = append(buf, fileInfoOut...)

	n, err := c.Stream.Write(buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("write size is not equal to buf size")
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", n, "bytes")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fileBuf := bufio.NewReader(file)
	num, err := io.Copy(c.Stream, fileBuf)
	if err != nil {
		return err
	}
	if num != osFileInfo.Size() {
		return fmt.Errorf("write size is not equal to file size")
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "sent", num, "bytes")
	}

	return nil
}
