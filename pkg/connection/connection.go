package connection

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/quic-go/quic-go"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	"github.com/quic-s/quics-protocol/pkg/utils/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
	"google.golang.org/protobuf/proto"
)

type Connection struct {
	logLevel int
	writeMut *sync.Mutex
	Conn     quic.Connection
	Stream   quic.Stream
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
		logLevel: logLevel,
		writeMut: &sync.Mutex{},
		Conn:     conn,
		Stream:   stream,
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
		log.Println("quics-protocol: ", header.Type)
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
		log.Println("quics-protocol: ", message.Type, string(message.Data))
	}
	return message, nil
}

func (c *Connection) ReadFile() (string, *fileinfo.FileInfo, io.Reader, error) {
	fileInfoSizeBuf := make([]byte, 2)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "read file info size")
	}
	n, err := io.ReadFull(c.Stream, fileInfoSizeBuf)
	if err != nil {
		return "", nil, nil, err
	}
	if n != 2 {
		return "", nil, nil, fmt.Errorf("file info size is not 2 bytes")
	}
	fileInfoSize := uint16(binary.BigEndian.Uint16(fileInfoSizeBuf))

	fileInfoBuf := make([]byte, fileInfoSize)
	n, err = io.ReadFull(c.Stream, fileInfoBuf)
	if err != nil {
		return "", nil, nil, err
	}
	if n != int(fileInfoSize) {
		return "", nil, nil, fmt.Errorf("file info size is not %d bytes", fileInfoSize)
	}

	protoFileInfo := &pb.FileInfo{}
	proto.Unmarshal(fileInfoBuf, protoFileInfo)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", protoFileInfo.Type)
	}

	fileInfo, err := fileinfo.DecodeFileInfo(protoFileInfo.FileInfo)
	if err != nil {
		fmt.Println(err)
		return "", nil, nil, err
	}
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", fileInfo.Name, fileInfo.Size, "bytes")
		log.Println("quics-protocol: ", "read file")
	}

	fileReader := io.LimitReader(c.Stream, fileInfo.Size)
	if c.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "init file reader with size", fileInfo.Size)
	}

	return protoFileInfo.Type, fileInfo, fileReader, nil
}

func (c *Connection) SendMessage(msgType string, data []byte) error {
	c.writeMut.Lock()
	err := c.writeHeader(pb.MessageType_MESSAGE)
	if err != nil {
		return err
	}

	err = c.writeMessage(msgType, data)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) SendFile(fileType string, filePath string) error {
	c.writeMut.Lock()
	err := c.writeHeader(pb.MessageType_FILE)
	if err != nil {
		return err
	}

	err = c.writeFile(fileType, filePath)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) SendFileWithMessage(fileMsgType string, data []byte, filePath string) error {
	c.writeMut.Lock()
	err := c.writeHeader(pb.MessageType_FILE_W_MESSAGE)
	if err != nil {
		return err
	}

	err = c.writeMessage(fileMsgType, data)
	if err != nil {
		return err
	}

	err = c.writeFile(fileMsgType, filePath)
	if err != nil {
		return err
	}
	c.writeMut.Unlock()

	return nil
}

func (c *Connection) writeHeader(messageType pb.MessageType) error {
	header := &pb.Header{
		Type: messageType,
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

func (c *Connection) writeMessage(msgType string, data []byte) error {
	message := &pb.Message{
		Type: msgType,
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

func (c *Connection) writeFile(fileType string, filePath string) error {
	osFileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	bFileInfo, err := fileinfo.EncodeFileInfo(osFileInfo)
	if err != nil {
		return err
	}

	fileInfo := &pb.FileInfo{
		Type:     fileType,
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

	num, err := io.Copy(c.Stream, file)
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
