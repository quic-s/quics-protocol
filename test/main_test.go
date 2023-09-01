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

func TestServerClientMessage(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

		// initialize server
		quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
		if err != nil {
			log.Println("quics-protocol: ", err)
		}

		// goroutine for closing server after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			quicServer.Close()
		}()

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
	}()

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
}

func TestServerClientFile(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

		// initialize server
		quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
		if err != nil {
			log.Println("quics-protocol: ", err)
		}

		// goroutine for closing server after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			quicServer.Close()
		}()

		err = quicServer.RecvFileHandleFunc("test", func(conn *qp.Connection, fileType string, fileInfo *qp.FileInfo, fileReader io.Reader) {
			log.Println("quics-protocol: ", "file received ", fileInfo.Name)
			file, err := os.Create("received.txt")
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			n, err := io.Copy(file, fileReader)
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			if n != fileInfo.Size {
				log.Println("quics-protocol: ", "file size is not equal to received size")
				return
			}
			log.Println("quics-protocol: ", "file saved")
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
	}()

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
}

func TestServerClientFileWithMessage(t *testing.T) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()

		// initialize server
		quicServer, err := qp.New(qp.LOG_LEVEL_INFO)
		if err != nil {
			log.Println("quics-protocol: ", err)
		}

		// goroutine for closing server after 5 seconds
		go func() {
			time.Sleep(5 * time.Second)
			quicServer.Close()
		}()

		err = quicServer.RecvFileMessageHandleFunc("test", func(conn *qp.Connection, fileMsgType string, data []byte, fileInfo *qp.FileInfo, fileReader io.Reader) {
			log.Println("quics-protocol: ", "message received ", conn.Conn.RemoteAddr().String())
			log.Println("quics-protocol: ", fileMsgType, string(data))

			log.Println("quics-protocol: ", "file received ", fileInfo.Name)
			file, err := os.Create("received.txt")
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			n, err := io.Copy(file, fileReader)
			if err != nil {
				log.Println("quics-protocol: ", err)
				return
			}
			if n != fileInfo.Size {
				log.Println("quics-protocol: ", "file size is not equal to received size")
				return
			}
			log.Println("quics-protocol: ", "file saved")
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
	}()

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
}
