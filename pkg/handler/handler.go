package handler

import (
	"context"
	"fmt"
	"log"

	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	qpStream "github.com/quic-s/quics-protocol/pkg/stream"
)

type Handler struct {
	logLevel           int
	ctx                context.Context
	cancel             context.CancelFunc
	transactionHandler map[string]func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte)
}

func New(loglevel int, ctx context.Context, cancel context.CancelFunc) *Handler {
	transactionHandler := make(map[string]func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte))

	transactionHandler["default"] = func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, data []byte) {
		log.Println("quics-protocol: 'default' handler is not set")
	}

	return &Handler{
		logLevel:           loglevel,
		ctx:                ctx,
		cancel:             cancel,
		transactionHandler: transactionHandler,
	}
}

func (h *Handler) RouteTransaction(conn *qpConn.Connection) error {
	for {
		stream, err := conn.Conn.AcceptStream(h.ctx)
		if err != nil {
			log.Println("quics-protocol: ", err)
			return err
		}
		if h.logLevel <= qpLog.INFO {
			log.Println("quics-protocol: ", "stream accepted")
		}

		go func() {
			newStream, err := qpStream.New(h.logLevel, stream)
			if err != nil {
				log.Println("quics-protocol: ", err)
				newStream.Close()
				return
			}
			transaction, err := qpConn.RecvTransactionHandshake(newStream)
			if err != nil {
				log.Println("quics-protocol: ", err)
				newStream.Close()
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
				h.transactionHandler["default"](conn, newStream, transaction.TransactionName, transaction.TransactionID)
			} else {
				h.transactionHandler[transaction.TransactionName](conn, newStream, transaction.TransactionName, transaction.TransactionID)
			}
		}()
	}
}

func (h *Handler) AddTransactionHandleFunc(transactionName string, handler func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, transactionID []byte)) error {
	if transactionName == "default" {
		return fmt.Errorf("quics-protocol: 'default' is reserved transaction name")
	}
	h.transactionHandler[transactionName] = handler
	return nil
}

func (h *Handler) DefaultTransactionHandleFunc(handler func(conn *qpConn.Connection, stream *qpStream.Stream, transactionName string, transactionID []byte)) error {
	h.transactionHandler["default"] = handler
	return nil
}
