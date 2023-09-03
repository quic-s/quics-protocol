package qp

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpHandler "github.com/quic-s/quics-protocol/pkg/handler"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	"github.com/quic-s/quics-protocol/pkg/utils/fileinfo"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

type QP struct {
	ctx          context.Context
	cancel       context.CancelFunc
	quicConf     *quic.Config
	quicListener *quic.Listener
	handler      *qpHandler.Handler
	logLevel     int
}

func New(logLevel int) (*QP, error) {
	ctx, cancel := context.WithCancel(context.Background())
	quicConf := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	}
	handler := qpHandler.New()

	return &QP{
		ctx:          ctx,
		cancel:       cancel,
		quicConf:     quicConf,
		quicListener: nil,
		handler:      handler,
		logLevel:     logLevel,
	}, nil
}

func (q *QP) Dial(address *net.UDPAddr, tlsConf *tls.Config) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	conn, err := quic.Dial(q.ctx, udpConn, address, tlsConf, q.quicConf)
	if err != nil {
		return nil, err
	}

	stream, err := conn.OpenStreamSync(q.ctx)
	if err != nil {
		return nil, err
	}

	newConn, err := qpConn.New(q.logLevel, conn, stream)
	if err != nil {
		return nil, err
	}
	return newConn, nil
}

func (q *QP) DialWithMessage(address *net.UDPAddr, tlsConf *tls.Config, msgType string, data []byte) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	conn, err := quic.Dial(q.ctx, udpConn, address, tlsConf, q.quicConf)
	if err != nil {
		return nil, err
	}

	stream, err := conn.OpenStreamSync(q.ctx)
	if err != nil {
		return nil, err
	}

	newConn, err := qpConn.New(q.logLevel, conn, stream)
	if err != nil {
		return nil, err
	}

	err = newConn.SendMessage(msgType, data)
	if err != nil {
		return nil, err
	}
	return newConn, nil
}

func (q *QP) Listen(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *Connection)) error {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp4", address)
	if err != nil {
		return err
	}

	q.quicListener, err = quic.Listen(udpConn, tlsConf, q.quicConf)
	if err != nil {
		return err
	}

	for {
		conn, err := q.quicListener.Accept(q.ctx)
		if err != nil {
			if err.Error() == ConnectionClosedByPeer || err == io.EOF {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Connection closed by peer")
				}
				return err
			}
			if err.Error() == quic.ErrServerClosed.Error() {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Server closed")
				}
				return err
			}
			if err.Error() == context.Canceled.Error() {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Context canceled")
				}
				return err
			}
			log.Println("quics-protocol: ", err)
			return err
		}
		if q.logLevel <= LOG_LEVEL_INFO {
			log.Println("quics-protocol: ", "conn accepted")
		}

		go func(c quic.Connection) {
			stream, err := c.AcceptStream(q.ctx)
			if err != nil {
				if err.Error() == ConnectionClosedByPeer || err == io.EOF {
					if q.logLevel <= LOG_LEVEL_INFO {
						log.Println("quics-protocol: ", "Connection closed by peer")
					}
					return
				}
				log.Println("quics-protocol: ", err)
				return
			}
			if q.logLevel <= LOG_LEVEL_INFO {
				log.Println("quics-protocol: ", "stream accepted")
			}

			newConn, err := qpConn.New(q.logLevel, conn, stream)
			if err != nil {
				return
			}

			connHandler(newConn)
			q.handler.RouteConnection(newConn)
		}(conn)
	}
}

func (q *QP) ListenWithMessage(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *Connection, msgType string, data []byte)) error {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp4", address)
	if err != nil {
		return err
	}

	q.quicListener, err = quic.Listen(udpConn, tlsConf, q.quicConf)
	if err != nil {
		return err
	}

	for {
		conn, err := q.quicListener.Accept(q.ctx)
		if err != nil {
			if err.Error() == ConnectionClosedByPeer || err == io.EOF {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Connection closed by peer")
				}
				return err
			}
			if err.Error() == quic.ErrServerClosed.Error() {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Server closed")
				}
				return err
			}
			if err.Error() == context.Canceled.Error() {
				if q.logLevel <= LOG_LEVEL_INFO {
					log.Println("quics-protocol: ", "Context canceled")
				}
				return err
			}
			log.Println("quics-protocol: ", err)
			return err
		}
		if q.logLevel <= LOG_LEVEL_INFO {
			log.Println("quics-protocol: ", "conn accepted")
		}

		go func(c quic.Connection) {
			stream, err := c.AcceptStream(q.ctx)
			if err != nil {
				if err.Error() == ConnectionClosedByPeer || err == io.EOF {
					if q.logLevel <= LOG_LEVEL_INFO {
						log.Println("quics-protocol: ", "Connection closed by peer")
					}
					return
				}
				log.Println("quics-protocol: ", err)
				return
			}
			if q.logLevel <= LOG_LEVEL_INFO {
				log.Println("quics-protocol: ", "stream accepted")
			}

			newConn, err := qpConn.New(q.logLevel, conn, stream)
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}

			header, err := newConn.ReadHeader()
			if header.Type != pb.MessageType_MESSAGE {
				log.Println("quics-protocol: ", "Not message type")
				return
			}
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			msg, err := newConn.ReadMessage()
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			connHandler(newConn, msg.Type, msg.Data)
			q.handler.RouteConnection(newConn)
		}(conn)
	}
}

func (q *QP) Close() error {
	if q.quicListener != nil {
		if q.logLevel <= LOG_LEVEL_INFO {
			log.Println("quics-protocol: ", "Close quicListener")
		}
		err := q.quicListener.Close()
		if err != nil {
			return err
		}
	}
	q.cancel()
	return nil
}

func (q *QP) RecvMessageHandleFunc(msgType string, handler func(conn *Connection, msgType string, data []byte)) error {
	q.handler.AddMessageHandleFunc(msgType, handler)
	return nil
}

func (q *QP) RecvFileHandleFunc(fileType string, handler func(conn *Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	q.handler.AddFileHandleFunc(fileType, handler)
	return nil
}

func (q *QP) RecvFileMessageHandleFunc(fileMsgType string, handler func(conn *Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error {
	q.handler.AddFileMessageHandleFunc(fileMsgType, handler)
	return nil
}

func (q *QP) RecvMessage(handler func(conn *Connection, msgType string, data []byte)) error {
	q.handler.DefaultMessageHandleFunc(handler)
	return nil
}

func (q *QP) RecvFile(handler func(conn *Connection, msgType string, data []byte)) error {
	q.handler.DefaultMessageHandleFunc(handler)
	return nil
}

func (q *QP) RecvFileMessage(handler func(conn *Connection, msgType string, msgData []byte)) error {
	q.handler.DefaultMessageHandleFunc(handler)
	return nil
}
