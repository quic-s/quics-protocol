package qp

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-s/quics-protocol/pkg/connection"
	qpHandler "github.com/quic-s/quics-protocol/pkg/handler"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
)

type QP struct {
	ctx          context.Context
	cancel       context.CancelFunc
	quicConf     *quic.Config
	quicListener *quic.Listener
	handler      *qpHandler.Handler
	logLevel     int
}

/*
 * Create new quics-protocol instance with log level (LOG_LEVEL_DEBUG, LOG_LEVEL_INFO, LOG_LEVEL_ERROR)
 */
func New(logLevel int) (*QP, error) {
	ctx, cancel := context.WithCancel(context.Background())
	quicConf := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	}
	handler := qpHandler.New(logLevel, ctx, cancel)

	return &QP{
		ctx:          ctx,
		cancel:       cancel,
		quicConf:     quicConf,
		quicListener: nil,
		handler:      handler,
		logLevel:     logLevel,
	}, nil
}

/*
 * Dial to server with address and tls config.
 * Return connection instance and error.
 * Need to set receive handler before dialing.
 */
func (q *QP) Dial(address *net.UDPAddr, tlsConf *tls.Config) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	conn, err := quic.Dial(q.ctx, udpConn, address, tlsConf, q.quicConf)
	if err != nil {
		return nil, err
	}

	newConn, err := connection.New(q.logLevel, conn)
	if err != nil {
		return nil, err
	}

	go func() {
		err := q.handler.RouteTransaction(newConn)
		if err != nil {
			log.Println("quics-protocol: ", err)
			return
		}
	}()
	return newConn, nil
}

/*
 * Dial to server with address, tls config and initial message.
 * Server need to start with ListenWithMessage for handling initial message.
 * Return connection instance and error.
 * Need to set receive handler before dialing.
 */
func (q *QP) DialWithTransaction(address *net.UDPAddr, tlsConf *tls.Config, transactionName string, transactionFunc func(stream *Stream, transactionName string, transactionID []byte) error) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	conn, err := quic.Dial(q.ctx, udpConn, address, tlsConf, q.quicConf)
	if err != nil {
		return nil, err
	}

	newConn, err := connection.New(q.logLevel, conn)
	if err != nil {
		return nil, err
	}

	err = newConn.OpenTransaction(transactionName, transactionFunc)
	if err != nil {
		return nil, err
	}

	go q.handler.RouteTransaction(newConn)
	return newConn, nil
}

/*
 * Start server with address and tls config.
 * Return error.
 * Need to set receive handler before listening.
 */
func (q *QP) Listen(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *Connection)) error {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp", address)
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
			log.Println("quics-protocol: ", err)
			return err
		}
		if q.logLevel <= LOG_LEVEL_INFO {
			log.Println("quics-protocol: ", "conn accepted")
		}

		newConn, err := connection.New(q.logLevel, conn)
		if err != nil {
			return err
		}

		go q.handler.RouteTransaction(newConn)
		connHandler(newConn)
	}
}

/*
 * Close quics-protocol instance.
 */
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

func (q *QP) RecvTransactionHandleFunc(transactionName string, callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte)) error {
	err := q.handler.AddTransactionHandleFunc(transactionName, callback)
	if err != nil {
		return err
	}
	return nil
}

func (q *QP) DefaultRecvTransactionHandleFunc(callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte)) error {
	err := q.handler.DefaultTransactionHandleFunc(callback)
	if err != nil {
		return err
	}
	return nil
}
