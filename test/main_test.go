package main_test

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	qp "github.com/quic-s/quics-protocol"
)

var wg = sync.WaitGroup{}

func TestServerClient(t *testing.T) {
	// initialize server
	quicServer, err := initializeServer(t)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

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
	}()

	wg.Add(3)
	t.Run("Send Message to Server", func(t *testing.T) {
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
	})

	t.Run("Send File to Server", func(t *testing.T) {
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
		if err != nil {
			log.Println("quics-protocol: ", err)
		}

		// send message to server
		conn.SendFile("test", "test.txt")

		// delay for waiting message sent to server
		time.Sleep(3 * time.Second)
		conn.Close()
	})

	t.Run("Send File with Message to Server", func(t *testing.T) {
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
		if err != nil {
			log.Println("quics-protocol: ", err)
		}

		// send message to server
		conn.SendFileWithMessage("test", []byte("test message"), "test.txt")

		// delay for waiting message sent to server
		time.Sleep(3 * time.Second)
		conn.Close()
	})

	quicServer.Close()
}

func initializeServer(t *testing.T) (*qp.QP, error) {
	// initialize server
	quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
	if err != nil {
		return nil, err
	}
	err = quicServer.RecvMessageHandleFunc("test", func(conn *qp.Connection, msgType string, data []byte) {
		defer wg.Done()
		log.Println("quics-protocol: ", "message received ", conn.Conn.RemoteAddr().String())
		log.Println("quics-protocol: ", msgType, string(data))
	})
	if err != nil {
		return nil, err
	}

	err = quicServer.RecvFileHandleFunc("test", func(conn *qp.Connection, fileType string, fileInfo *qp.FileInfo, fileReader io.Reader) {
		defer wg.Done()
		log.Println("quics-protocol: ", "file received ", fileInfo.Name)
		file, err := os.Create("received.txt")
		if err != nil {
			t.Fatal(err)
		}
		n, err := io.Copy(file, fileReader)
		if err != nil {
			t.Fatal(err)
		}
		if n != fileInfo.Size {
			t.Fatalf("quics-protocol: read only %dbytes", n)
		}
		log.Println("quics-protocol: ", "file saved with ", n, "bytes")
	})
	if err != nil {
		return nil, err
	}

	err = quicServer.RecvFileMessageHandleFunc("test", func(conn *qp.Connection, fileMsgType string, data []byte, fileInfo *qp.FileInfo, fileReader io.Reader) {
		defer wg.Done()
		log.Println("quics-protocol: ", "message received ", conn.Conn.RemoteAddr().String())
		log.Println("quics-protocol: ", fileMsgType, string(data))

		log.Println("quics-protocol: ", "file received ", fileInfo.Name)
		file, err := os.Create("received2.txt")
		if err != nil {
			t.Fatal(err)
		}
		n, err := io.Copy(file, fileReader)
		if err != nil {
			t.Fatal(err)
		}
		if n != fileInfo.Size {
			log.Println("quics-protocol: ", "read only ", n, "bytes")
			t.Fatal(err)
		}
		log.Println("quics-protocol: ", "file saved")
	})
	if err != nil {
		return nil, err
	}

	return quicServer, nil
}
