# quics-protocol

**quics-protocol** is a simple experimental protocol for sending and receiving bytes meesage or file over QUIC protocol.

It uses the [quic-go](https://github.com/quic-go/quic-go) library to implement QUIC protocol communication, which aims to achieve faster and more reliable connections.

## Usage

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

