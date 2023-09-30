package main

import (
	"crypto/tls"
	"log"
	"net"

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

		err := stream.SendBMessage([]byte("test message"))
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
		return nil
	})

	conn.Close()
}
