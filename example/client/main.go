package main

import (
	"log"

	qp "github.com/quic-s/quics-protocol"
)

func main() {
	quic, err := qp.New()
	if err != nil {
		log.Panicln(err)
	}

	err = quic.Dial("localhost:18080")
	if err != nil {
		log.Panicln(err)
	}
	defer quic.Close()

	quic.SendMessage("test", []byte("test message"))
}
