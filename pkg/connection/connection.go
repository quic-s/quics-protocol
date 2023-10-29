package connection

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	qpStream "github.com/quic-s/quics-protocol/pkg/stream"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

// Connection is a connection instance that is created when a client connects to a server.
type Connection struct {
	logLevel int
	Conn     quic.Connection
}

// New creates a new connection instance.
// This method is used internally by quics-protocol.
// So, you may don't need to use it directly.
func New(logLevel int, conn quic.Connection) (*Connection, error) {
	if conn == nil {
		return nil, errors.New("conn is nil")
	}
	if !conn.ConnectionState().TLS.HandshakeComplete {
		return nil, errors.New("TLS handshake is not completed")
	}

	return &Connection{
		logLevel: logLevel,
		Conn:     conn,
	}, nil
}

// Close closes the connection.
func (c *Connection) Close() error {
	if c == nil || c.Conn == nil {
		return errors.New("connection instance is nil")
	}
	err := c.Conn.CloseWithError(0, "Connection closed by peer")
	if err != nil {
		return err
	}
	return nil
}

// CloseWithError closes the connection with an error message.
func (c *Connection) CloseWithError(message string) error {
	if c == nil || c.Conn == nil {
		return errors.New("connection instance is nil")
	}
	err := c.Conn.CloseWithError(0, message)
	if err != nil {
		return err
	}
	return nil
}

// OpenTransaction opens a transaction to the server.
// The transaction name and transaction function are needed as parameters.
// The transaction name is used to determine which handler to use on the receiving side.
// `transactionFunc“ is called when the transaction is opened.
// The stream, transaction name, and transaction id are passed as parameters.
// The stream is used to send and receive messages and files.
func (c *Connection) OpenTransaction(transactionName string, transactionFunc func(stream *qpStream.Stream, transactionName string, transactionID []byte) error) error {
	if c == nil || c.Conn == nil {
		return errors.New("connection instance is nil")
	}

	stream, err := c.Conn.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}
	newStream, err := qpStream.New(c.logLevel, stream)
	if err != nil {
		newStream.SendError(err.Error())
		return err
	}
	defer newStream.Close()

	transactionID, err := uuid.New().MarshalBinary()
	if err != nil {
		newStream.SendError(err.Error())
		return err
	}

	err = TransactionHandshake(newStream, transactionName, transactionID)
	if err != nil {
		newStream.SendError(err.Error())
		return err
	}

	err = transactionFunc(newStream, transactionName, transactionID)
	if err != nil {
		newStream.SendError(err.Error())
		return err
	}

	return nil
}

// TransactionHandshake sends a transaction request to the server when a transaction is opened.
// This method is used internally when opening a transaction.
// So, you may don't need to use it directly.
func TransactionHandshake(stream *qpStream.Stream, transactionName string, transactionID []byte) error {
	requestId, err := uuid.New().MarshalBinary()
	if err != nil {
		return err
	}
	err = qpStream.WriteHeader(stream, pb.RequestType_TRANSACTION, requestId, "")
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
		return errors.New("quics-protocol: Not transaction type")
	}
	transaction, err := qpStream.ReadTransaction(stream)
	if err != nil {
		return err
	}
	if transaction.TransactionName != transactionName {
		return errors.New("quics-protocol: Transaction name is not matched")
	}
	if string(transaction.TransactionID) != string(transactionID) {
		return errors.New("quics-protocol: Transaction ID is not matched")
	}
	return nil
}

// RecvTransactionHandshake receives a transaction request from the client when a transaction is opened.
// This method is used internally when opening a transaction.
// So, you may don't need to use it directly.
func RecvTransactionHandshake(stream *qpStream.Stream) (*pb.Transaction, error) {
	header, err := qpStream.ReadHeader(stream)
	if err != nil {
		return nil, err
	}
	if header.RequestType != pb.RequestType_TRANSACTION {
		return nil, errors.New("quics-protocol: Not transaction type")
	}
	transaction, err := qpStream.ReadTransaction(stream)
	if err != nil {
		return nil, err
	}

	err = qpStream.WriteHeader(stream, pb.RequestType_TRANSACTION, transaction.TransactionID, "")
	if err != nil {
		return nil, err
	}
	err = qpStream.WriteTransaction(stream, transaction.TransactionName, transaction.TransactionID)
	if err != nil {
		return nil, err
	}

	return transaction, nil
}
