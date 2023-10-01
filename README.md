# quics-protocol

**quics-protocol** is a simple experimental protocol for sending and receiving bytes meesage or file over QUIC protocol.

It uses the [quic-go](https://github.com/quic-go/quic-go) library to implement QUIC protocol communication, which aims to achieve faster and more reliable connections.

[Features](#features) | [Usage](#usage) | [Types](#types) | [Design](#design) | [Contribute](#contribute)

## Features

- Open 

## Usage

### Install and Import

First, you need to get quics-protocol package.

```bash
go get github.com/quic-s/quics-protocol
```

Then, import it in your code.

```go
import qp "github.com/quic-s/quics-protocol"
```

**quics-protocol** is a library for communication between a server and a client. The communication is initiated by opening a port on the server using the Listen method and dialing on the client.

For easy to use, import as qp is recommended.

Example code for server and client is as follows. Also see [`/test`](https://github.com/quic-s/quics-protocol/tree/main/test) directory is helpful.

### Server

```go
package main

import (
	"crypto/tls"
	"log"
	"net"

	qp "github.com/quic-s/quics-protocol"
)

func main() {
	// initialize server
	quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
	if err != nil {
		log.Println("quics-server: ", err)
	}

	err = quicServer.RecvTransactionHandleFunc("test", func(conn *qp.Connection, stream *qp.Stream, transactionName string, transactionID []byte) {
		log.Println("quics-server: ", "message received ", conn.Conn.RemoteAddr().String())

		data, err := stream.RecvBMessage()
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}
		log.Println("quics-server: ", "recv message from client")
		log.Println("quics-server: ", "message: ", string(data))
		if string(data) != "send message" {
			log.Println("quics-server: Recieved message is not inteded message.")
			return
		}

		err = stream.SendBMessage([]byte("return message"))
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}

		fileInfo, fileContent, err := stream.RecvFile()
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}
		log.Println("quics-server: ", "file received")

		err = fileInfo.WriteFileWithInfo("example/server/received.txt", fileContent)
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}
		log.Println("quics-server: ", "file saved")
	})
	if err != nil {
		log.Println("quics-server: ", err)
	}

	cert, err := qp.GetCertificate("", "")
	if err != nil {
		log.Println("quics-server: ", err)
		return
	}
	tlsConf := &tls.Config{
		Certificates: cert,
		NextProtos:   []string{"quics-protocol"},
	}
	// start server
	quicServer.Listen(&net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 18080}, tlsConf, func(conn *qp.Connection) {
		log.Println("quics-server: ", "new connection ", conn.Conn.RemoteAddr().String())
	})
}
```

### Client

```go
package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"

	qp "github.com/quic-s/quics-protocol"
)

func main() {
	// initialize client
	quicClient, err := qp.New(qp.LOG_LEVEL_INFO)
	if err != nil {
		log.Println("quics-protocol: ", err)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quics-protocol"},
	}
	// start client
	conn, err := quicClient.Dial(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 18080}, tlsConf)
	if err != nil {
		log.Println("quics-client: ", err)
	}

	log.Println("quics-client: ", "send message to server")
	// send message to server
	conn.OpenTransaction("test", func(stream *qp.Stream, transactionName string, transactionID []byte) error {
		log.Println("quics-client: ", "send transaction to server")
		log.Println("quics-client: ", "transactionName: ", transactionName)
		log.Println("quics-client: ", "transactionID: ", string(transactionID))

		err := stream.SendBMessage([]byte("send message"))
		if err != nil {
			log.Println("quics-client: ", err)
			return err
		}

		data, err := stream.RecvBMessage()
		if err != nil {
			log.Println("quics-client: ", err)
			return err
		}
		log.Println("quics-client: ", "recv message from server")
		log.Println("quics-client: ", "message: ", string(data))
		if string(data) != "return message" {
			return fmt.Errorf("quics-client: Received message is not the intended message")
		}

		log.Println("quics-client: ", "send file to server")
		err = stream.SendFile("test/test/test.txt")
		if err != nil {
			log.Println("quics-client: ", err)
			return err
		}

		log.Println("quics-client: ", "transaction finished")
		return nil
	})

	// wait for all stream is sent to server
	time.Sleep(3 * time.Second)
	conn.Close()
}
```

## Types

* [QP](#qp)
	* [New](#new)
	* [Listen](#listen)
	* [Dial](#dial)
	* [DialWithTransaction](#dialwithtransaction)
	* [Close](#close)
* [Connection](#connection)
	* [New](#new-1)
	* [OpenTransaction](#opentransaction)
	* [Close](#close-1)
	* [CloseWithError](#closewitherror)
* [Stream](#stream)
	* [New](#new-2)
	* [SendMessage](#sendmessage)
	* [SendFile](#sendfile)
	* [SendFileBMessage](#sendfilebmessage)
	* [RecvBMessage](#recvbmessage)
	* [RecvFile](#recvfile)
	* [RecvFileBMessage](#recvfilebmessage)
	* [Close](#close-2)
* [FileInfo](#fileinfo)
	* [WriteFileWithInfo](#writefilewithinfo)
	* [ToProtobuf](#toprotobuf)

### QP

```go
type QP struct {
	ctx          context.Context
	cancel       context.CancelFunc
	quicConf     *quic.Config
	quicListener *quic.Listener
	handler      *qpHandler.Handler
	logLevel     int
}
```

QP is a quics-protocol instance.

### Methods

#### New

```go
func New(logLevel int) (*qp.QP, error)
```

New creates a new quics-protocol instance. logLevel is used to set the log level that quics-protocol uses internally. The log level is set to qp.LOG_LEVEL_INFO by default. 

logLevel can be set to one of the following values.

```go
const (
	LOG_LEVEL_DEBUG = iota
	LOG_LEVEL_INFO
	LOG_LEVEL_ERROR
)
```

#### Listen

```go
func (q *QP) Listen(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *qp.Connection)) error
```

Listen starts a server listening for incoming connections on the UDP address addr with TLS configuration tlsConf.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive transactions from the client after establish connections, use RecvTransactionHandleFunc.)

#### Dial

```go
func (qp *QP) Dial(address *net.UDPAddr, tlsConf *tls.Config) (*qp.Connection, error)
```

Dial connects to the address addr on the named network net with TLS configuration tlsConf.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive transactions from the client after establish connections, use RecvTransactionHandleFunc.)

#### DialWithTransaction

```go
func (q *QP) DialWithTransaction(address *net.UDPAddr, tlsConf *tls.Config, transactionName string, transactionFunc func(stream *Stream, transactionName string, transactionID []byte) error) (*Connection, error) 
```

DialWithTransaction connects to the address addr on the named network net with TLS configuration tlsConf. Unlike Dial, this method also opens a transaction to the server. So, the transaction name and transaction function are needed as parameters.

This can be used to send authentication information and more to the server in a transaction when connecting to the server. 

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive transactions from the client after establish connections, use RecvTransactionHandleFunc.)

#### Close

```go
func (q *QP) Close() error
```

Close closes the quics-protocol instance.

#### RecvTransactionHandleFunc

```go
func (q *QP) RecvTransactionHandleFunc(transactionName string, callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte)) error
```

RecvTransactionHandleFunc sets the handler function for receiving transactions from the client. The transaction name and callback function are needed as parameters. The transaction name is used to determine which handler to use on the receiving side.

#### DefaultRecvTransactionHandleFunc

```go
func (q *QP) DefaultRecvTransactionHandleFunc(callback func(conn *Connection, stream *Stream, transactionName string, transactionID []byte)) error
```

DefaultRecvTransactionHandleFunc sets the default handler function for receiving transactions from the client. The callback function is needed as a parameter. The default handler is used when the transaction name is not set or the transaction name is not found.

### Connection

```go
type Connection struct {
	logLevel int
	Conn     quic.Connection
}
```

Connection is a connection instance that is created when a client connects to a server.

### Methods

#### New

```go
func New(logLevel int, conn quic.Connection, stream quic.Stream) (*Connection, error)
```

New creates a new connection instance. This method is used internally by quics-protocol. So, you may don't need to use it directly.

#### OpenTransaction

```go
func (c *Connection) OpenTransaction(transactionName string, transactionFunc func(stream *qpStream.Stream, transactionName string, transactionID []byte) error) error
```

OpenTransaction opens a transaction to the server. The transaction name and transaction function are needed as parameters. The transaction name is used to determine which handler to use on the receiving side.

transactionFunc is called when the transaction is opened. The stream, transaction name, and transaction id are passed as parameters. The stream is used to send and receive messages and files.

#### Close

```go
func (c *Connection) Close() error
```

Close closes the connection.

#### CloseWithError

```go
func (c *Connection) CloseWithError(message string) error
```

CloseWithError closes the connection with an error message.

### Stream

```go
type Stream struct {
	logLevel int
	Stream   quic.Stream
}
```

Stream is a stream instance that is created when a transaction is opened. Below is a list of methods that can be used with the stream. You can send and receive messages and files multiple times within a single transaction. 

> **Important Note!!**: **Sending and receiving** methods are must be used in **pairs**. If you send a message, you must receive a message. If you send a file, you must receive a file.
This is because the receiving side is waiting for a request from the sending side. If you don't send a request, the receiving side will wait forever. **Please check the example code for how to use it.**

> Note: Stream is closed automatically when the transaction is closed. So, you may don't need to close it directly.

### Methods

#### New

```go
func New(logLevel int, stream quic.Stream) (*Stream, error)
```

New creates a new stream instance. This method is used internally by quics-protocol. So, you may don't need to use it directly.

#### SendMessage

```go
func (s *Stream) SendBMessage(data []byte) error
```

SendBMessage sends a bytes message through the connection. The message data needs to be passed as a parameter. This method must be used in pairs with RecvBMessage.

#### SendFile

```go
func (s *Stream) SendFile(filePath string) error
```

SendFile sends a file through the connection. The file path needs to be passed as a parameter. The metadata of the file is automatically sent to the receiving side. If the filePath is a directory, the directory is sent as a file.  This method must be used in pairs with RecvFile.

#### SendFileBMessage

```go
func (s *Stream) SendFileBMessage(data []byte, filePath string) error
```

SendFileBMessage sends a file with bytes message through the connection. The message data and file path need to be passed as parameters. The metadata of the file is automatically sent to the receiving side. If the filePath is a directory, the directory is sent as a file. This method must be used in pairs with RecvFileBMessage.

#### RecvBMessage

```go
func (s *Stream) RecvBMessage() ([]byte, error)
```

RecvBMessage receives a bytes message through the connection. The message data is returned as a result. This method must be used in pairs with SendBMessage.

#### RecvFile

```go
func (s *Stream) RecvFile() (*fileinfo.FileInfo, io.Reader, error)
```

RecvFile receives a file through the connection. The file metadata and file data are returned as a result. This method must be used in pairs with SendFile.

> Note: The file data is returned as an io.Reader type object. So, you must read this to receive the file. If you don't read it, the receiving side will wait forever.

> Tip: You can use the [WriteFileWithInfo](#writefilewithinfo) method to wrtie the file with metadata to the disk. See the example code for more details.

#### RecvFileBMessage

```go
func (s *Stream) RecvFileBMessage() ([]byte, *fileinfo.FileInfo, io.Reader, error)
```

RecvFileBMessage receives a file with bytes message through the connection. The message data, file metadata, and file data are returned as a result. This method must be used in pairs with SendFileBMessage.

> Note: The file data is returned as an io.Reader type object. So, you must read this to receive the file. If you don't read it, the receiving side will wait forever.

> Tip: You can use the [WriteFileWithInfo](#writefilewithinfo) method to wrtie the file with metadata to the disk. See the example code for more details.

#### Close

```go
func (c *Connection) Close() error
```

Close closes the stream. Stream is closed automatically when the transaction is closed. So, you may don't need to use it directly. 

### FileInfo

```go
type FileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}
```

FileInfo is a file metadata structure. It is used to send and receive file metadata through the connection. You can get this struct as a result when receiving a file through the connection.

### Methods

#### WriteFileWithInfo

```go
func (f *FileInfo) WriteFileWithInfo(filePath string, fileContent io.Reader) error
```

WriteFileWithInfo writes the file with metadata to the disk. The file path and file data(io.Reader type) need to be passed as parameters.

This method creates a directory if the directory does not exist or received file is directory. If the file already exists, it will be overwritten.

#### ToProtobuf

```go
func (f *FileInfo) ToProtobuf() (*pb.FileInfo, error)
```

ToProtobuf converts the FileInfo to protobuf format. This method is used internally by quics-protocol. So, you may don't need to use it directly.

## Design

**quics-protocol** largely consists of quics-protocol, connection, stream, and handler. The quics-protocol is a library for communication between a server and a client. The communication is initiated by opening a port on the server using the Listen method and dialing on the client. 

The connection is a connection instance that is created when a client connects to a server. After connection is established, connection instance is passed to the handler's RouteTransaction method. 

The stream is a stream instance that is created when a transaction is opened. The stream is used to send and receive messages and files. Sending and receiving methods are must be used in pairs.

The handler is created when quics-protocol instance is created. It is used to handle messages and files received from the client. But handler is only used internally by quics-protocol. So, you may don't need to use it directly.

### Transaction and Request

**quics-protocol** uses the concept of transactions and requests. The transaction is a frame of streams in the QUIC protocol that tracks the byte range of each stream separately. This allows multiple transactions to be sent over a single connection in parallel, and requests to be sent and received synchronously within a single transaction.

The request is the unit of sending a message or file over a transaction (stream). Multiple requests can be sent or received within a single transaction, but the sender and receiver must be committed to the order in which they are sent and received. If the order is not guaranteed, the receiving side will wait forever. 

Below is a diagram of the transaction and request that used in [usage example code](#usage).

![transaction](https://github.com/quic-s/quics-protocol/assets/20539422/841509e5-72a3-404f-81ca-4a22aaebe5b9)

### protocol data structure

Every message and file sent through quics-protocol has a header. The header is used to determine which handler to use on the receiving side. 

Google's protobuf library is used to design protocol data. The protocol data structure is as follows.

```protobuf
message Header {
    RequestType requestType = 1;
    bytes requestId = 2;
}

enum RequestType {
    UNKNOWN = 0;
    TRANSACTION = 1;
    // BMESSAGE means a bytes message
    BMESSAGE = 2;
    FILE = 3;
    FILE_BMESSAGE = 4;
}

message Transaction {
    string transactionName = 1;
    bytes transactionID = 2;
}

message FileInfo {
    string name = 1;
    int64 size = 2;
    int32 mode = 3;
    bytes modTime = 4;
    bool isDir = 5;
}
```

When data is actually transmitted through the quic protocol, it is transmitted as a byte stream in the form below. 

- BMessage

```
 0                   1
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Header Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            Header             \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Message Length        |
|           (32 bits)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            BMessage           \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Above structure is case of sending message.

The header length word is 16 bits. It indicates the length of the following Header message. The Header is in protocol buffer format. 

The Header describes the request type, and request id. The request type is specified the data structure(bmessage, file, or file with bmessage). 

Message length word is 32 bits. It indicates the length of the following BMessage message. So, the maximum size of the bmessage is 4GB.

The BMessage is just bytes data. So, users need to serialize and deserialize the data.

- File

```
 0                   1
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Header Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            Header             \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         File Info Length      |
|           (16 bits)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\           File Info           \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            File               \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The header length and header are the same as the message above. 

However, the content after the header is slightly different, first followed by the 16-bit file info size. The file info is protocol buffer format with bytes data, so it needs to be serialized and deserialized as form of qp.FileInfo.

After the file info, the file data is followed. File data is byte data equal to the size transmitted through fileinfo above. 

Because the file can be large, it is passed as a parameter to the handler function as an io.Reader type object. Users can read this and receive the file.

- File with bmessage

```
 0                   1
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Header Length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            Header             \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Message Length        |
|           (32 bits)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            BMessage           \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         File Info Length      |
|           (16 bits)           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\           File Info           \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
/                               /
\            File               \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The file with message is the combination form of message and file. The header and message are the same as the message above. 

The file info and file are the same as the file above.

It can be seen as simply a form in which messages and files are delivered at once as a transaction.

## Contribute

To report bugs or request features, please use the issue tracker. Before you do so, make sure you are running the latest version, and please do a quick search to see if the issue has already been reported.

For more discussion, please join the [quics discord](https://discord.com/invite/HRtY7pNZz2)