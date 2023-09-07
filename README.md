# quics-protocol

**quics-protocol** is a simple experimental protocol for sending and receiving bytes meesage or file over QUIC protocol.

It uses the [quic-go](https://github.com/quic-go/quic-go) library to implement QUIC protocol communication, which aims to achieve faster and more reliable connections.

[Features](#features) | [Usage](#usage) | [Types](#types) | [Design](#design) | [Contribute](#contribute)

## Features

- Send and receive messages
- Send and receive files
- Send and receive messages and files
- Send and receive messages with response
- Send and receive files with response
- Send and receive messages and files with response

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
		log.Println("quics-protocol: ", err)
	}

	err = quicServer.RecvMessageHandleFunc("test", func(conn *qp.Connection, msgType string, data []byte) {
		log.Println("quics-protocol: ", "message received ", conn.Conn.RemoteAddr().String())
		log.Println("quics-protocol: ", msgType, string(data))
	})
	if err != nil {
		log.Println("quics-protocol: ", err)
	}

	cert, err := qp.GetCertificate("", "")
	if err != nil {
		log.Println("quics-protocol: ", err)
		return
	}
	tlsConf := &tls.Config{
		Certificates: cert,
		NextProtos:   []string{"quics-protocol"},
	}
	// start server
	quicServer.Listen(&net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 18080}, tlsConf, func(conn *qp.Connection) {
		log.Println("quics-protocol: ", "new connection ", conn.Conn.RemoteAddr().String())
	})
}
```

### Client

```go
package main

import (
	"crypto/tls"
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
		log.Println("quics-protocol: ", err)
	}

	// send message to server
	conn.SendMessage("test", []byte("test message"))

	// delay for waiting message sent to server
	time.Sleep(3 * time.Second)
	conn.Close()
}
```

## Types

* [QP](#qp)
	* [New](#new)
	* [Listen](#listen)
	* [Dial](#dial)
	* [ListenWithMessage](#listenwithmessage)
	* [DialWithMessage](#dialwithmessage)
	* [Close](#close)
	* [RecvMessageHandleFunc](#recvmessagehandlefunc)
	* [RecvFileHandleFunc](#recvfilehandlefunc)
	* [RecvFileMessageHandleFunc](#recvfilemessagehandlefunc)
	* [RecvMessageWithResponseHandleFunc](#recvmessagewithresponsehandlefunc)
	* [RecvFileWithResponseHandleFunc](#recvfilewithresponsehandlefunc)
	* [RecvFileMessageWithResponseHandleFunc](#recvfilemessagewithresponsehandlefunc)
	* [RecvMessage](#recvmessage)
	* [RecvFile](#recvfile)
	* [RecvFileMessage](#recvfilemessage)
	* [RecvMessageWithResponse](#recvmessagewithresponse)
	* [RecvFileWithResponse](#recvfilewithresponse)
	* [RecvFileMessageWithResponse](#recvfilemessagewithresponse)
* [Connection](#connection)
	* [New](#new-1)
	* [SendMessage](#sendmessage)
	* [SendFile](#sendfile)
	* [SendFileMessage](#sendfilemessage)
	* [SendMessageWithResponse](#sendmessagewithresponse)
	* [SendFileWithResponse](#sendfilewithresponse)
	* [SendFileMessageWithResponse](#sendfilemessagewithresponse)
	* [Close](#close-1)
	* [CloseWithError](#closewitherror)

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

New creates a new quics-protocol instance.

#### Listen

```go
func (q *QP) Listen(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *qp.Connection)) error
```

Listen starts a server listening for incoming connections on the UDP address addr with TLS configuration tlsConf.

> Note: This method is paired with Dial, so clients should use it to initiate communication. Not DialWithMessage.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive messages from the client after establish connections, use RecvMessageHandleFunc.)

#### Dial

```go
func (qp *QP) Dial(address *net.UDPAddr, tlsConf *tls.Config) (*qp.Connection, error)
```

Dial connects to the address addr on the named network net with TLS configuration tlsConf.

> Note: This method is paired with Listen, so servers should use it to initiate communication. Not ListenWithMessage.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive messages from the client after establish connections, use RecvMessageHandleFunc.)

#### ListenWithMessage

```go
func (q *QP) ListenWithMessage(address *net.UDPAddr, tlsConf *tls.Config, connHandler func(conn *qp.Connection, msgType string, data []byte)) error
```

ListenWithMessage starts a server listening for incoming connections on the UDP address addr with TLS configuration tlsConf. Unlike Listen, this method also receives messages from the client when establishing connections.

This can be used to implement handlers such as authentication when first establishing communication with a user.This method is used to receive messages from the client when establish connections. The message type and data are passed to the handler. 

> Note: This method is paired with DialWithMessage, so clients should use it to initiate communication.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive messages from the client after establish connections, use RecvMessageHandleFunc.)

#### DialWithMessage

```go
func (q *QP) DialWithMessage(address *net.UDPAddr, tlsConf *tls.Config, msgType string, data []byte) (*qp.Connection, error)
```

DialWithMessage connects to the address addr on the named network net with TLS configuration tlsConf. Unlike Dial, this method also sends messages to the server when establishing connections.

This can be used to send authentication information and more to the server in a message the moment you first open a connection. So, the message type and data are passed to the handler.

> Note: This method is paired with ListenWithMessage, so servers should use it to initiate communication.

> Note: Receiving handler must be set before calling this method. (ex: If you want to receive messages from the client after establish connections, use RecvMessageHandleFunc.)

#### Close

```go
func (q *QP) Close() error
```

Close closes the quics-protocol instance.

#### RecvMessageHandleFunc

```go
func (q *QP) RecvMessageHandleFunc(msgType string, handler func(conn *qp.Connection, msgType string, data []byte)) error
```

RecvMessageHandleFunc sets the handler function for receiving messages from the client.

#### RecvFileHandleFunc

```go
func (q *QP) RecvFileHandleFunc(fileType string, handler func(conn *Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error
```

RecvFileHandleFunc sets the handler function for receiving files from the client.

#### RecvFileMessageHandleFunc

```go
func (q *QP) RecvFileMessageHandleFunc(fileMsgType string, handler func(conn *Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader)) error
```

RecvFileMessageHandleFunc sets the handler function for receiving files from the client. Unlike RecvFileHandleFunc, this method also receives messages from the client when receiving files.

#### RecvMessageWithResponseHandleFunc

```go
func (q *QP) RecvMessageWithResponseHandleFunc(msgType string, handler func(conn *Connection, msgType string, data []byte) []byte) error
```

RecvMessageWithResponseHandleFunc sets the handler function for receiving messages from the client. Unlike RecvMessageHandleFunc, this method also sends a response to the client.

#### RecvFileWithResponseHandleFunc

```go
func (q *QP) RecvFileWithResponseHandleFunc(fileType string, handler func(conn *Connection, fileType string, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error
```

RecvFileWithResponseHandleFunc sets the handler function for receiving files from the client. Unlike RecvFileHandleFunc, this method also sends a response to the client.

#### RecvFileMessageWithResponseHandleFunc

```go
func (q *QP) RecvFileMessageWithResponseHandleFunc(fileMsgType string, handler func(conn *Connection, fileMsgType string, msgData []byte, fileInfo *fileinfo.FileInfo, fileReader io.Reader) []byte) error
```

RecvFileMessageWithResponseHandleFunc sets the handler function for receiving files from the client. Unlike RecvFileMessageHandleFunc, this method also sends a response to the client.

#### RecvMessage

```go
func (q *QP) RecvMessage(handler func(conn *Connection, msgType string, data []byte)) error
```

RecvMessage sets the default handler function for receiving messages from the client. 

#### RecvFile

```go
func (q *QP) RecvFile(handler func(conn *Connection, msgType string, data []byte)) error
```

RecvFile sets the default handler function for receiving files from the client.

#### RecvFileMessage

```go
func (q *QP) RecvFileMessage(handler func(conn *Connection, msgType string, msgData []byte)) error
```

RecvFileMessage sets the default handler function for receiving files from the client. Unlike RecvFile, this method also receives messages from the client when receiving files.

#### RecvMessageWithResponse

```go
func (q *QP) RecvMessageWithResponse(handler func(conn *Connection, msgType string, data []byte) []byte) error
```

RecvMessageWithResponse sets the default handler function for receiving messages from the client. Unlike RecvMessage, this method also sends a response to the client.

#### RecvFileWithResponse

```go
func (q *QP) RecvFileWithResponse(handler func(conn *Connection, msgType string, data []byte) []byte) error
```

RecvFileWithResponse sets the default handler function for receiving files from the client. Unlike RecvFile, this method also sends a response to the client.

#### RecvFileMessageWithResponse

```go
func (q *QP) RecvFileMessageWithResponse(handler func(conn *Connection, msgType string, msgData []byte) []byte) error
```

RecvFileMessageWithResponse sets the default handler function for receiving files from the client. Unlike RecvFileMessage, this method also sends a response to the client.

### Connection

```go
type Connection struct {
	logLevel            int
	writeMut            *sync.Mutex
	MsgResponseChan     map[string]chan []byte
	FileResponseChan    map[string]chan []byte
	FileMsgResponseChan map[string]chan []byte
	Conn                quic.Connection
	Stream              quic.Stream
}
```

Connection is a connection instance that is created when a client connects to a server.

### Methods

#### New

```go
func New(logLevel int, conn quic.Connection, stream quic.Stream) (*Connection, error)
```

New creates a new connection instance. This method is used internally by quics-protocol. So, you may don't need to use it directly.

#### SendMessage

```go
func (c *Connection) SendMessage(msgType string, data []byte) error
```

SendMessage sends a message through the connection. The message type and data need to be passed as parameters. The message type is used to determine which handler to use on the receiving side.

#### SendFile

```go
func (c *Connection) SendFile(fileType string, filePath string) error
```

SendFile sends a file through the connection. The file type and file path need to be passed as parameters. The file type is used to determine which handler to use on the receiving side.

#### SendFileMessage

```go
func (c *Connection) SendFileMessage(fileMsgType string, data []byte, filePath string) error
```

SendFileMessage sends a file and message through the connection. The fileMsgType, message data, and file path need to be passed as parameters. The fileMsgType is used to determine which handler to use on the receiving side.

#### SendMessageWithResponse

```go
func (c *Connection) SendMessageWithResponse(msgType string, data []byte) ([]byte, error)
```

SendMessageWithResponse sends a message through the connection. The message type and data need to be passed as parameters. The message type is used to determine which handler to use on the receiving side. Unlike SendMessage, this method also waits for a response from the receiving side.

#### SendFileWithResponse

```go
func (c *Connection) SendFileWithResponse(fileType string, filePath string) ([]byte, error)
```

SendFileWithResponse sends a file through the connection. The file type and file path need to be passed as parameters. The file type is used to determine which handler to use on the receiving side. Unlike SendFile, this method also waits for a response from the receiving side.

#### SendFileMessageWithResponse

```go
func (c *Connection) SendFileMessageWithResponse(fileMsgType string, data []byte, filePath string) ([]byte, error)
```

SendFileMessageWithResponse sends a file and message through the connection. The fileMsgType, message data, and file path need to be passed as parameters. The fileMsgType is used to determine which handler to use on the receiving side. Unlike SendFileMessage, this method also waits for a response from the receiving side.

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

## Design

**quics-protocol** largely consists of quics-protocol, connection, and handler. The quics-protocol is a library for communication between a server and a client. The communication is initiated by opening a port on the server using the Listen method and dialing on the client. 

The connection is a connection instance that is created when a client connects to a server. After connection is established, connection instance is passed to the handler's RouteConnection method. 

The handler is created when quics-protocol instance is created. It is used to handle messages and files received from the client. But handler is only used internally by quics-protocol. So, you may don't need to use it directly.

### protocol data structure

Every message and file sent through quics-protocol has a header. The header is used to determine which handler to use on the receiving side. 

Google's protobuf library is used to design protocol data. The protocol data structure is as follows.

```protobuf
message Header {
    MessageType messageType = 1;
    string requestType = 2;
    bytes requestId = 3;
}

enum MessageType {
    UNKNOWN = 0;
    MESSAGE = 1;
    FILE = 2;
    FILE_MESSAGE = 3;
    MESSAGE_W_RESPONSE = 4;
    FILE_W_RESPONSE = 5;
    FILE_MESSAGE_W_RESPONSE = 6;
    RESPONSE_MESSAGE = 7;
    RESPONSE_FILE = 8;
    RESPONSE_FILE_MESSAGE = 9;
}

message Message {
    bytes data = 1;
}

message FileInfo {
    bytes fileInfo = 1;
}

message Response {
    bytes data = 1;
}
```

When data is actually transmitted through the quic protocol, it is transmitted as a byte stream in the form below. 

- Message

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
\            Message            \
/                               /
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Above structure is case of sending message.

The header length word is 16 bits. It indicates the length of the following Header message. The Header is in protocol buffer format. 

The Header describes the message type, request type, and request id. The message type is specified the data structure(message, file, or file with message). 

The request type is specified the handler to use on the receiving side. The request id is used to match the response from the receiving side.

Message length word is 32 bits. It indicates the length of the following Message message. So, the maximum size of the message is 4GB.

The Message is just bytes data. So, user's need to serialize and deserialize the data.

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

- File with message

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
\            Message            \
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

- Response

Response is a same form as message. The only difference is that the message type is different.

Because the response is just bytes data, it need to be serialized and deserialized by user.

## Contribute

To report bugs or request features, please use the issue tracker. Before you do so, make sure you are running the latest version, and please do a quick search to see if the issue has already been reported.

For more discussion, please join the [quics discord](https://discord.com/invite/HRtY7pNZz2)