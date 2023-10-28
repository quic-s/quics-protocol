package qp

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-s/quics-protocol/pkg/connection"
	qpHandler "github.com/quic-s/quics-protocol/pkg/handler"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
)

// QP is a quics-protocol instance.
// To create a new instance, use the New method.
// For more information, see the README.md of the quics-protocol repository.
// https://github.com/quic-s/quics-protocol
type QP struct {
	ctx          context.Context
	cancel       context.CancelFunc
	quicConf     *quic.Config
	quicListener *quic.Listener
	handler      *qpHandler.Handler
	logLevel     int
}

// Create new quics-protocol instance with log level (LOG_LEVEL_DEBUG, LOG_LEVEL_INFO, LOG_LEVEL_ERROR)
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

// Dial connects to the address addr on the named network net with TLS configuration tlsConf.
// Return connection instance and error.
// Need to set receive handler using RecvTransactionHandleFunc method before dialing.
func (q *QP) Dial(host string, port int, tlsConf *tls.Config) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp6", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) < 1 {
		return nil, errors.New("wrong domain name")
	}
	if q.logLevel <= LOG_LEVEL_INFO {
		log.Println("quics-protocol: looked up ips ", ips)
	}

	address := &net.UDPAddr{
		IP:   ips[0],
		Port: port,
	}
	if q.logLevel <= LOG_LEVEL_INFO {
		log.Println("quics-protocol: dial to ", address)
	}

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

// DialWithTransaction connects to the address addr on the named network net with TLS configuration tlsConf.
// Unlike Dial, this method also opens a transaction to the server. So, the transaction name and transaction function are needed as parameters.
// This method is paired with ListenWithTransaction. So, you must use ListenWithTransaction on the server side.
// Return connection instance and error.
// Need to set receive handler using RecvTransactionHandleFunc before dialing.
func (q *QP) DialWithTransaction(host string, port int, tlsConf *tls.Config, transactionName string, transactionFunc func(stream *Stream, transactionName string, transactionID []byte) error) (*Connection, error) {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	if len(ips) < 1 {
		return nil, errors.New("wrong domain name")
	}
	if q.logLevel <= LOG_LEVEL_INFO {
		log.Println("quics-protocol: looked up ips ", ips)
	}

	address := &net.UDPAddr{
		IP:   ips[0],
		Port: port,
	}
	if q.logLevel <= LOG_LEVEL_INFO {
		log.Println("quics-protocol: dial to ", address)
	}

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

// Listen starts a server listening for incoming connections on the UDP address addr with TLS configuration tlsConf.
// Return error.
// Need to set receive handler using RecvTransactionHandleFunc method before listening.
func (q *QP) Listen(address string, tlsConf *tls.Config, connHandler func(conn *Connection)) error {
	if q.logLevel == LOG_LEVEL_DEBUG {
		q.quicConf.Tracer = qpLog.NewQLogTracer()
	}

	udpAddr, err := net.ResolveUDPAddr("udp6", address)
	if err != nil {
		return err
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
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

// ListenWithTransaction starts a server listening for incoming connections on the UDP address addr with TLS configuration tlsConf.
// Unlike Listen, this method also receives initial transactions from the client. So, the transaction function are needed as parameters.
// This method is paired with DialWithTransaction. So, you must use DialWithTransaction on the client side.
// Return error.
// Need to set receive handler using RecvTransactionHandleFunc before listening.
func (q *QP) ListenWithTransaction(address *net.UDPAddr, tlsConf *tls.Config, transactionFunc func(conn *Connection, stream *Stream, transactionName string, transactionID []byte) error) error {
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
			newConn.CloseWithError(err.Error())
			return err
		}
		go func() {
			stream, err := q.handler.RecvTransaction(newConn)
			if err != nil {
				log.Println("quics-protocol: ", err)
				err := stream.Close()
				if err != nil {
					log.Println("quics-protocol: ", err)
					newConn.CloseWithError(err.Error())
				}
				return
			}

			transaction, err := connection.RecvTransactionHandshake(stream)
			if err != nil {
				log.Println("quics-protocol: ", err)
				err := stream.Close()
				if err != nil {
					log.Println("quics-protocol: ", err)
					newConn.CloseWithError(err.Error())
				}
				return
			}
			if q.logLevel <= qpLog.INFO {
				log.Println("quics-protocol: ", "transaction accepted")
			}

			err = transactionFunc(newConn, stream, transaction.TransactionName, transaction.TransactionID)
			if err != nil {
				log.Println("quics-protocol: ", err)
				newConn.CloseWithError(err.Error())
				return
			}

			go q.handler.RouteTransaction(newConn)
		}()
	}
}

// Close quics-protocol instance.
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

// RecvTransactionHandleFunc sets the handler function for receiving transactions from the client.
// The transaction name and callback function are needed as parameters.
// The transaction name is used to determine which handler to use on the receiving side.
func (q *QP) RecvTransactionHandleFunc(transactionName string, callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte) error) error {
	err := q.handler.AddTransactionHandleFunc(transactionName, callback)
	if err != nil {
		return err
	}
	return nil
}

// DefaultRecvTransactionHandleFunc sets the default handler function for receiving transactions from the client.
// The callback function is needed as a parameter.
// The default handler is used when the transaction name is not set or the transaction name is not found.
func (q *QP) DefaultRecvTransactionHandleFunc(callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte) error) error {
	err := q.handler.DefaultTransactionHandleFunc(callback)
	if err != nil {
		return err
	}
	return nil
}

// GetErrChan returns the error channel of the quics-protocol instance.
// This channel is used to receive errors when errors occur in the receive transaction handler function.
// This is optional. If you do not need to receive errors, you do not need to use this channel.
func (q *QP) GetErrChan() chan error {
	return q.handler.GetErrChan()
}
