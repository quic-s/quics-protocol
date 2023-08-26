package main_test

import (
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	qp "github.com/quic-s/quics-protocol"
	pb "github.com/quic-s/quics-protocol/proto/v1"
)

func TestServerClientMessage(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

		// initialize server
		quicServer, err := qp.New()
		if err != nil {
			log.Panicln(err)
		}

		// goroutine for closing server after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			quicServer.Close()
		}()

		err = quicServer.RecvMessage(func(conn quic.Connection, message *pb.Message) {
			log.Println("message received ", conn.RemoteAddr().String())
			log.Println(message.Type, string(message.Message))
		})
		if err != nil {
			log.Panicln(err)
		}
		// start server
		err = quicServer.Listen("0.0.0.0", 18080)
		if err != nil {
			log.Panicln(err)
		}
	}()

	// initialize client
	quicClient, err := qp.New()
	if err != nil {
		log.Panicln(err)
	}

	// start client
	err = quicClient.Dial("localhost:18080")
	if err != nil {
		log.Panicln(err)
	}

	// send message to server
	quicClient.SendMessage("test", []byte("test message"))

	// delay for waiting message sent to server
	time.Sleep(3 * time.Second)
	quicClient.Close()
}

func TestServerClientFile(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

		// initialize server
		quicServer, err := qp.New()
		if err != nil {
			log.Panicln(err)
		}

		// goroutine for closing server after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			quicServer.Close()
		}()

		err = quicServer.RecvFile(func(conn quic.Connection, fileInfo *pb.FileInfo, fileBuf []byte) {
			log.Println("file received ", conn.RemoteAddr().String(), fileInfo.Path)
			os.WriteFile("received.txt", fileBuf, 0644)
		})
		if err != nil {
			log.Panicln(err)
		}
		// start server
		err = quicServer.Listen("0.0.0.0", 18080)
		if err != nil {
			log.Panicln(err)
		}
	}()

	// initialize client
	quicClient, err := qp.New()
	if err != nil {
		log.Panicln(err)
	}

	// start client
	err = quicClient.Dial("localhost:18080")
	if err != nil {
		log.Panicln(err)
	}

	file, err := os.ReadFile("test.txt")
	if err != nil {
		log.Panicln(err)
	}
	// send message to server
	quicClient.SendFile("test.txt", file)

	// delay for waiting message sent to server
	time.Sleep(3 * time.Second)
	quicClient.Close()
}
