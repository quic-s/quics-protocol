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

	err = quic.Listen("0.0.0.0", 18080)
	if err != nil {
		log.Panicln(err)
	}
	defer quic.Close()
}
