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
