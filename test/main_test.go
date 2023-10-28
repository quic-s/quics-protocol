package main_test

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	qp "github.com/quic-s/quics-protocol"
)

var wg = sync.WaitGroup{}

func TestServerClient(t *testing.T) {
	// initialize and run server
	wg.Add(2)
	quicServer, err := runServer(t)
	if err != nil {
		t.Fatal(err)
	}
	defer wg.Wait()

	wg.Add(1)
	t.Run("Send Message to Server", func(t *testing.T) {
		defer wg.Done()
		// initialize client
		quicClient, err := qp.New(qp.LOG_LEVEL_INFO)
		if err != nil {
			log.Println("quics-client: ", err)
		}

		tlsConf := &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"quics-protocol"},
		}
		// start client
		conn, err := quicClient.DialWithTransaction("localhost", 18080, tlsConf, "test", func(stream *qp.Stream, transactionName string, transactionID []byte) error {
			data, err := stream.RecvBMessage()
			if err != nil {
				log.Println("quics-client: ", err)
				return err
			}
			log.Println("quics-client: ", "recv message from client")
			log.Println("quics-client: ", "message: ", string(data))
			if string(data) != "send message" {
				log.Println("quics-client: Recieved message is not inteded message.")
				return err
			}

			err = stream.SendBMessage([]byte("return message"))
			if err != nil {
				log.Println("quics-client: ", err)
				return err
			}
			return nil
		})

		if err != nil {
			log.Println("quics-client: ", err)
		}

		// open transaction
		err = conn.OpenTransaction("test", func(stream *qp.Stream, transactionName string, transactionID []byte) error {
			log.Println("quics-client: ", "send transaction to server")
			log.Println("quics-client: ", "transactionName: ", transactionName)
			log.Println("quics-client: ", "transactionID: ", string(transactionID))

			err := stream.SendBMessage([]byte("send message"))
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
			if string(data) != "return message" {
				return fmt.Errorf("quics-client: Received message is not the intended message.")
			}

			log.Println("quics-client: ", "send file to server")
			err = stream.SendFile("test/test.txt")
			if err != nil {
				log.Println("quics-client: ", err)
				return err
			}

			log.Println("quics-client: ", "transaction finished")
			return nil
		})
		if err != nil {
			log.Println("quics-client: ", err)
		}

		// wait for all stream is sent to server
		time.Sleep(3 * time.Second)
		conn.Close()
	})

	quicServer.Close()
}

func runServer(t *testing.T) (*qp.QP, error) {
	// initialize server
	quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
	if err != nil {
		return nil, err
	}
	err = quicServer.RecvTransactionHandleFunc("test", func(conn *qp.Connection, stream *qp.Stream, transactionName string, transactionID []byte) error {
		defer wg.Done()
		log.Println("quics-server: ", "message received ", conn.Conn.RemoteAddr().String())

		data, err := stream.RecvBMessage()
		if err != nil {
			log.Println("quics-server: ", err)
			return err
		}
		log.Println("quics-server: ", "recv message from client")
		log.Println("quics-server: ", "message: ", string(data))
		if string(data) != "send message" {
			log.Println("quics-server: Recieved message is not inteded message.")
			return err
		}

		err = stream.SendBMessage([]byte("return message"))
		if err != nil {
			log.Println("quics-server: ", err)
			return err
		}

		fileInfo, fileContent, err := stream.RecvFile()
		if err != nil {
			log.Println("quics-server: ", err)
			return err
		}
		log.Println("quics-server: ", "file received")

		err = fileInfo.WriteFileWithInfo("received/test.txt", fileContent)
		if err != nil {
			log.Println("quics-server: ", err)
			return err
		}
		log.Println("quics-server: ", "file saved")
		return nil
	})
	if err != nil {
		return nil, err
	}

	go func() {
		defer wg.Done()

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
		quicServer.ListenWithTransaction(":18080", tlsConf, func(conn *qp.Connection, stream *qp.Stream, transactionName string, transactionID []byte) error {
			log.Println("quics-server: ", "transactionName: ", transactionName)
			log.Println("quics-server: ", "transactionID: ", string(transactionID))

			err := stream.SendBMessage([]byte("send message"))
			if err != nil {
				log.Println("quics-server: ", err)
				return err
			}

			data, err := stream.RecvBMessage()
			if err != nil {
				log.Println("quics-server: ", err)
				return err
			}
			log.Println("quics-server: ", "recv message from client")
			log.Println("quics-server: ", "message: ", string(data))
			if string(data) != "return message" {
				return fmt.Errorf("quics-server: Received message is not the intended message.")
			}
			return nil
		})
	}()
	return quicServer, nil
}
