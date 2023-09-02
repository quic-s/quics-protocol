package handler

import (
	"io"
	"log"

	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpErr "github.com/quic-s/quics-protocol/pkg/error"
	"github.com/quic-s/quics-protocol/pkg/utils/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

type Handler struct {
	messageHandler     map[string]func(conn *qpConn.Connection, msgType string, data []byte)
	fileHandler        map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)
	fileMessageHandler map[string]func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)

	defaultMessageHandler     func(conn *qpConn.Connection, msgType string, data []byte)
	defaultFileHandler        func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)
	defaultFileMessageHandler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)
}

func New() *Handler {
	return &Handler{
		messageHandler:     make(map[string]func(conn *qpConn.Connection, msgType string, data []byte)),
		fileHandler:        make(map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)),
		fileMessageHandler: make(map[string]func(conn *qpConn.Connection, fileMsgType string, data []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)),
		defaultMessageHandler: func(conn *qpConn.Connection, msgType string, data []byte) {
			log.Println("quics-protocol: No default recevied message handler")
		},
		defaultFileHandler: func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) {
			log.Println("quics-protocol: No default recevied file handler")
		},
		defaultFileMessageHandler: func(conn *qpConn.Connection, fileMsgType string, data []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) {
			log.Println("quics-protocol: No default recevied file with message handler")
		},
	}
}

func (h *Handler) RouteConnection(conn *qpConn.Connection) {
	go func() {
		for {
			header, err := conn.ReadHeader()
			if err != nil {
				if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
					log.Println("quics-protocol: ", "Connection closed by peer")
					return
				}
				log.Println("quics-protocol: ", err)
				return
			}

			switch header.Type {
			case pb.MessageType_MESSAGE:
				message, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.messageHandler[message.Type]; exists {
					go h.messageHandler[message.Type](conn, message.Type, message.Data)
				} else {
					go h.defaultMessageHandler(conn, message.Type, message.Data)
				}
			case pb.MessageType_FILE:
				fileType, fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileHandler[fileType]; exists {
					go h.fileHandler[fileType](conn, fileType, fileInfo, fileReader)
				} else {
					go h.defaultFileHandler(conn, fileType, fileInfo, fileReader)
				}
			case pb.MessageType_FILE_W_MESSAGE:
				message, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				_, fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileMessageHandler[message.Type]; exists {
					go h.fileMessageHandler[message.Type](conn, message.Type, message.Data, fileInfo, fileReader)
				} else {
					go h.defaultFileMessageHandler(conn, message.Type, message.Data, fileInfo, fileReader)
				}
			default:
				log.Println("quics-protocol: ", "unknown message type")
			}
		}
	}()
}

func (h *Handler) AddMessageHandleFunc(msgType string, handler func(conn *qpConn.Connection, msgType string, data []byte)) error {
	h.messageHandler[msgType] = handler
	return nil
}

func (h *Handler) AddFileHandleFunc(fileType string, handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.fileHandler[fileType] = handler
	return nil
}

func (h *Handler) AddFileMessageHandleFunc(fileMsgType string, handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.fileMessageHandler[fileMsgType] = handler
	return nil
}

func (h *Handler) DefaultMessageHandleFunc(handler func(conn *qpConn.Connection, msgType string, data []byte)) error {
	h.defaultMessageHandler = handler
	return nil
}

func (h *Handler) DefaultFileHandleFunc(handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.defaultFileHandler = handler
	return nil
}

func (h *Handler) DefaultFileMessageHandleFunc(handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.defaultFileMessageHandler = handler
	return nil
}
