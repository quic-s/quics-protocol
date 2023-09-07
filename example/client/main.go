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

	log.Println("quics-protocol: ", "send message to server")
	// send message to server
	conn.SendMessage("test", []byte("test message"))

	// delay for waiting message sent to server
	time.Sleep(3 * time.Second)
	conn.Close()
}
