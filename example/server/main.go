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
