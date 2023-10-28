package handler

import (
	"context"
	"errors"
	"log"

	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	qpStream "github.com/quic-s/quics-protocol/pkg/stream"
)

type Handler struct {
	logLevel           int
	ctx                context.Context
	cancel             context.CancelFunc
	errChan            chan error
	transactionHandler map[string]func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte) error
}

func New(loglevel int, ctx context.Context, cancel context.CancelFunc) *Handler {
	transactionHandler := make(map[string]func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte) error)

	transactionHandler["default"] = func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte) error {
		log.Println("quics-protocol: 'default' handler is not set")
		return nil
	}

	return &Handler{
		logLevel:           loglevel,
		ctx:                ctx,
		cancel:             cancel,
		errChan:            nil,
		transactionHandler: transactionHandler,
	}
}

func (h *Handler) RouteTransaction(conn *qpConn.Connection) error {
	for {
		stream, err := h.RecvTransaction(conn)
		if err != nil {
			log.Println("quics-protocol: ", err)
			return err
		}
		go func() {
			transaction, err := qpConn.RecvTransactionHandshake(stream)
			if err != nil {
				log.Println("quics-protocol: ", err)
				err := stream.Close()
				if err != nil {
					log.Println("quics-protocol: ", err)
				}
				return
			}
			if h.logLevel <= qpLog.INFO {
				log.Println("quics-protocol: ", "transaction accepted")
			}
			if h.transactionHandler[transaction.TransactionName] == nil {
				log.Println("quics-protocol: ", "handler for transaction ", transaction.TransactionName, " is not set. Use 'default' handler.")
				err = h.transactionHandler["default"](conn, stream, transaction.TransactionName, transaction.TransactionID)
				if err != nil {
					log.Println("quics-protocol: ", err)
					if h.errChan != nil {
						h.errChan <- err
					}
					stream.Close()
				}
			} else {
				err = h.transactionHandler[transaction.TransactionName](conn, stream, transaction.TransactionName, transaction.TransactionID)
				if err != nil {
					log.Println("quics-protocol: ", err)
					if h.errChan != nil {
						h.errChan <- err
					}
					stream.Close()
				}
			}
		}()
	}
}

func (h *Handler) RecvTransaction(conn *qpConn.Connection) (*qpStream.Stream, error) {
	stream, err := conn.Conn.AcceptStream(h.ctx)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}
	if h.logLevel <= qpLog.INFO {
		log.Println("quics-protocol: ", "stream accepted")
	}
	newStream, err := qpStream.New(h.logLevel, stream)
	if err != nil {
		log.Println("quics-protocol: ", err)
		newStream.Close()
		return nil, err
	}

	return newStream, nil
}

func (h *Handler) GetErrChan() chan error {
	h.errChan = make(chan error)
	return h.errChan
}

func (h *Handler) AddTransactionHandleFunc(transactionName string, handler func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, transactionID []byte) error) error {
	if transactionName == "default" {
		return errors.New("quics-protocol: 'default' is reserved transaction name")
	}
	h.transactionHandler[transactionName] = handler
	return nil
}

func (h *Handler) DefaultTransactionHandleFunc(handler func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, transactionID []byte) error) error {
	h.transactionHandler["default"] = handler
	return nil
}
