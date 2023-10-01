package connection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	qpStream "github.com/quic-s/quics-protocol/pkg/stream"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

type Connection struct {
	logLevel int
	Conn     quic.Connection
}

func New(logLevel int, conn quic.Connection) (*Connection, error) {
	if conn == nil {
		return nil, fmt.Errorf("conn is nil")
	}
	if !conn.ConnectionState().TLS.HandshakeComplete {
		return nil, fmt.Errorf("TLS handshake is not completed")
	}

	return &Connection{
		logLevel: logLevel,
		Conn:     conn,
	}, nil
}

func (c *Connection) Close() error {
	err := c.Conn.CloseWithError(0, "Connection closed by peer")
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

func (c *Connection) OpenTransaction(transactionName string, transactionFunc func(stream *qpStream.Stream, transactionName string, transactionID []byte) error) error {
	stream, err := c.Conn.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	newStream, err := qpStream.New(c.logLevel, stream)
	if err != nil {
		newStream.Close()
		return err
	}

	transactionID, err := uuid.New().MarshalBinary()
	if err != nil {
		newStream.Close()
		return err
	}

	err = TransactionHandshake(newStream, transactionName, transactionID)
	if err != nil {
		newStream.Close()
		return err
	}

	err = transactionFunc(newStream, transactionName, transactionID)
	if err != nil {
		newStream.Close()
		return err
	}

	err = newStream.Close()
	if err != nil {
		return err
	}

	return nil
}

func TransactionHandshake(stream *qpStream.Stream, transactionName string, transactionID []byte) error {
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return err
	}
	err = qpStream.WriteHeader(stream, pb.RequestType_TRANSACTION, requestId)
	if err != nil {
		return err
	}

	err = qpStream.WriteTransaction(stream, transactionName, transactionID)
	if err != nil {
		return err
	}
	header, err := qpStream.ReadHeader(stream)
	if err != nil {
		return err
	}
	if header.RequestType != pb.RequestType_TRANSACTION {
		return fmt.Errorf("quics-protocol: Not transaction type")
	}
	transaction, err := qpStream.ReadTransaction(stream)
	if err != nil {
		return err
	}
	if transaction.TransactionName != transactionName {
		return fmt.Errorf("quics-protocol: Transaction name is not matched")
	}
	if string(transaction.TransactionID) != string(transactionID) {
		return fmt.Errorf("quics-protocol: Transaction ID is not matched")
	}
	return nil
}

func RecvTransactionHandshake(stream *qpStream.Stream) (*pb.Transaction, error) {
	header, err := qpStream.ReadHeader(stream)
	if err != nil {
		return nil, err
	}
	if header.RequestType != pb.RequestType_TRANSACTION {
		return nil, fmt.Errorf("quics-protocol: Not transaction type")
	}
	transaction, err := qpStream.ReadTransaction(stream)
	if err != nil {
		return nil, err
	}

	err = qpStream.WriteHeader(stream, pb.RequestType_TRANSACTION, transaction.TransactionID)
	if err != nil {
		return nil, err
	}
	err = qpStream.WriteTransaction(stream, transaction.TransactionName, transaction.TransactionID)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}
