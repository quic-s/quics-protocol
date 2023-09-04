package handler

import (
	"fmt"
	"io"
	"log"

	"github.com/google/uuid"
	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpErr "github.com/quic-s/quics-protocol/pkg/error"
	"github.com/quic-s/quics-protocol/pkg/utils/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

type Handler struct {
	messageHandler     map[string]func(conn *qpConn.Connection, msgType string, data []byte)
	fileHandler        map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)
	fileMessageHandler map[string]func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)

	messageWithResponseHandler     map[string]func(conn *qpConn.Connection, msgType string, data []byte) []byte
	fileWithResponseHandler        map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte
	fileMessageWithResponseHandler map[string]func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte
}

func New() *Handler {
	messageHandler := make(map[string]func(conn *qpConn.Connection, msgType string, data []byte))
	fileHandler := make(map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader))
	fileMessageHandler := make(map[string]func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader))

	messageWithResponseHandler := make(map[string]func(conn *qpConn.Connection, msgType string, data []byte) []byte)
	fileWithResponseHandler := make(map[string]func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte)
	fileMessageWithResponseHandler := make(map[string]func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte)

	messageHandler["default"] = func(conn *qpConn.Connection, msgType string, data []byte) {
		log.Println("quics-protocol: 'default' handler is not set")
	}
	fileHandler["default"] = func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) {
		log.Println("quics-protocol: 'default' handler is not set")
	}
	fileMessageHandler["default"] = func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) {
		log.Println("quics-protocol: 'default' handler is not set")
	}

	messageWithResponseHandler["default"] = func(conn *qpConn.Connection, msgType string, data []byte) []byte {
		log.Println("quics-protocol: 'default' handler is not set")
		return nil
	}
	fileWithResponseHandler["default"] = func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte {
		log.Println("quics-protocol: 'default' handler is not set")
		return nil
	}
	fileMessageWithResponseHandler["default"] = func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte {
		log.Println("quics-protocol: 'default' handler is not set")
		return nil
	}

	return &Handler{
		messageHandler:                 messageHandler,
		fileHandler:                    fileHandler,
		fileMessageHandler:             fileMessageHandler,
		messageWithResponseHandler:     messageWithResponseHandler,
		fileWithResponseHandler:        fileWithResponseHandler,
		fileMessageWithResponseHandler: fileMessageWithResponseHandler,
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

			switch header.MessageType {
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

				if _, exists := h.messageHandler[header.RequestType]; exists {
					h.messageHandler[header.RequestType](conn, header.RequestType, message.Data)
				} else {
					h.messageHandler["default"](conn, header.RequestType, message.Data)
				}
			case pb.MessageType_FILE:
				fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileHandler[header.RequestType]; exists {
					h.fileHandler[header.RequestType](conn, header.RequestType, fileInfo, fileReader)
				} else {
					h.fileHandler["default"](conn, header.RequestType, fileInfo, fileReader)
				}
			case pb.MessageType_FILE_MESSAGE:
				message, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileMessageHandler[header.RequestType]; exists {
					h.fileMessageHandler[header.RequestType](conn, header.RequestType, message.Data, fileInfo, fileReader)
				} else {
					h.fileMessageHandler["default"](conn, header.RequestType, message.Data, fileInfo, fileReader)
				}
			case pb.MessageType_MESSAGE_W_RESPONSE:
				message, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.messageWithResponseHandler[header.RequestType]; exists {
					response := h.messageWithResponseHandler[header.RequestType](conn, header.RequestType, message.Data)
					conn.SendResponse(pb.MessageType_RESPONSE_MESSAGE, header.RequestId, header.RequestType, response)
				} else {
					response := h.messageWithResponseHandler["default"](conn, header.RequestType, message.Data)
					conn.SendResponse(pb.MessageType_RESPONSE_MESSAGE, header.RequestId, header.RequestType, response)
				}
			case pb.MessageType_FILE_W_RESPONSE:
				fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileWithResponseHandler[header.RequestType]; exists {
					response := h.fileWithResponseHandler[header.RequestType](conn, header.RequestType, fileInfo, fileReader)
					conn.SendResponse(pb.MessageType_RESPONSE_FILE, header.RequestId, header.RequestType, response)
				} else {
					response := h.fileWithResponseHandler["default"](conn, header.RequestType, fileInfo, fileReader)
					conn.SendResponse(pb.MessageType_RESPONSE_FILE, header.RequestId, header.RequestType, response)
				}
			case pb.MessageType_FILE_MESSAGE_W_RESPONSE:
				message, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				fileInfo, fileReader, err := conn.ReadFile()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				if _, exists := h.fileMessageWithResponseHandler[header.RequestType]; exists {
					response := h.fileMessageWithResponseHandler[header.RequestType](conn, header.RequestType, message.Data, fileInfo, fileReader)
					conn.SendResponse(pb.MessageType_RESPONSE_FILE_MESSAGE, header.RequestId, header.RequestType, response)
				} else {
					response := h.fileMessageWithResponseHandler["default"](conn, header.RequestType, message.Data, fileInfo, fileReader)
					conn.SendResponse(pb.MessageType_RESPONSE_FILE_MESSAGE, header.RequestId, header.RequestType, response)
				}
			case pb.MessageType_RESPONSE_MESSAGE:
				response, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				requestUUID, err := uuid.FromBytes(header.RequestId)
				if err != nil {
					log.Println("quics-protocol: ", err)
					return
				}
				conn.MsgResponseChan[requestUUID.String()] <- response.Data
			case pb.MessageType_RESPONSE_FILE:
				response, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				requestUUID, err := uuid.FromBytes(header.RequestId)
				if err != nil {
					log.Println("quics-protocol: ", err)
					return
				}
				conn.FileResponseChan[requestUUID.String()] <- response.Data
			case pb.MessageType_RESPONSE_FILE_MESSAGE:
				response, err := conn.ReadMessage()
				if err != nil {
					if err.Error() == qpErr.ConnectionClosedByPeer || err == io.EOF {
						log.Println("quics-protocol: ", "Connection closed by peer")
						return
					}
					log.Println("quics-protocol: ", err)
					return
				}

				requestUUID, err := uuid.FromBytes(header.RequestId)
				if err != nil {
					log.Println("quics-protocol: ", err)
					return
				}
				conn.FileMsgResponseChan[requestUUID.String()] <- response.Data
			default:
				log.Println("quics-protocol: ", "unknown message type")
			}
		}
	}()
}

func (h *Handler) AddMessageHandleFunc(msgType string, handler func(conn *qpConn.Connection, msgType string, data []byte)) error {
	if msgType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved message type")
	}
	h.messageHandler[msgType] = handler
	return nil
}

func (h *Handler) AddFileHandleFunc(fileType string, handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	if fileType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved file type")
	}
	h.fileHandler[fileType] = handler
	return nil
}

func (h *Handler) AddFileMessageHandleFunc(fileMsgType string, handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	if fileMsgType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved file message type")
	}
	h.fileMessageHandler[fileMsgType] = handler
	return nil
}

func (h *Handler) AddMessageWithResponseHandleFunc(msgType string, handler func(conn *qpConn.Connection, msgType string, data []byte) []byte) error {
	if msgType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved message type")
	}
	h.messageWithResponseHandler[msgType] = handler
	return nil
}

func (h *Handler) AddFileWithResponseHandleFunc(fileType string, handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error {
	if fileType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved file type")
	}
	h.fileWithResponseHandler[fileType] = handler
	return nil
}

func (h *Handler) AddFileMessageWithResponseHandleFunc(fileMsgType string, handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error {
	if fileMsgType == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved file message type")
	}
	h.fileMessageWithResponseHandler[fileMsgType] = handler
	return nil
}

func (h *Handler) DefaultMessageHandleFunc(handler func(conn *qpConn.Connection, msgType string, data []byte)) error {
	h.messageHandler["default"] = handler
	return nil
}

func (h *Handler) DefaultFileHandleFunc(handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.fileHandler["default"] = handler
	return nil
}

func (h *Handler) DefaultFileMessageHandleFunc(handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	h.fileMessageHandler["default"] = handler
	return nil
}

func (h *Handler) DefaultMessageWithResponseHandleFunc(handler func(conn *qpConn.Connection, msgType string, data []byte) []byte) error {
	h.messageWithResponseHandler["default"] = handler
	return nil
}

func (h *Handler) DefaultFileWithResponseHandleFunc(handler func(conn *qpConn.Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error {
	h.fileWithResponseHandler["default"] = handler
	return nil
}

func (h *Handler) DefaultFileMessageWithResponseHandleFunc(handler func(conn *qpConn.Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error {
	h.fileMessageWithResponseHandler["default"] = handler
	return nil
}
