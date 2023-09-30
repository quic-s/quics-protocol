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
		log.Println("quics-server: ", "recv transaction from client")
		log.Println("quics-server: ", "transactionName: ", transactionName)
		log.Println("quics-server: ", "transactionID: ", string(transactionID))

		data, err := stream.RecvBMessage()
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}
		log.Println("quics-server: ", "recv message from client")
		log.Println("quics-server: ", "message: ", string(data))

		err = stream.SendBMessage([]byte("test message"))
		if err != nil {
			log.Println("quics-server: ", err)
			return
		}
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
