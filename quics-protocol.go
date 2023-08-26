package qp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-s/quics-protocol/pkg/qlog"
	pb "github.com/quic-s/quics-protocol/proto/v1"
	"google.golang.org/protobuf/proto"
)

const ConnectionClosedByPeer = "Application error 0x0 (remote): Connection closed by peer"
const QLog = false

type qp struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	tlsConf               *tls.Config
	quicConf              *quic.Config
	quicConn              quic.Connection
	quicListener          *quic.Listener
	quicStream            quic.Stream
	receiveFileHandler    func(conn quic.Connection, fileInfo *pb.FileInfo, fileBuf []byte)
	receiveMessageHandler func(conn quic.Connection, message *pb.Message)
}

func New() (*qp, error) {
	ctx, cancel := context.WithCancel(context.Background())
	quicConf := &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	}
	return &qp{
		ctx:          ctx,
		cancel:       cancel,
		tlsConf:      nil,
		quicConf:     quicConf,
		quicConn:     nil,
		quicListener: nil,
		quicStream:   nil,
	}, nil
}

func (q *qp) Dial(address string) error {
	if QLog {
		q.quicConf.Tracer = qlog.NewTracer()
	}
	q.tlsConf = &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quics-protocol"},
	}

	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		log.Panicln(err)
		return err
	}

	udpConn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		log.Panicln(err)
		return err
	}

	q.ctx, q.cancel = context.WithTimeout(q.ctx, 10*time.Second)
	q.quicConn, err = quic.Dial(q.ctx, udpConn, udpAddr, q.tlsConf, q.quicConf)
	if err != nil {
		log.Panicln(err)
		return err
	}
	q.quicStream, err = q.quicConn.OpenStreamSync(q.ctx)
	if err != nil {
		log.Panicln(err)
		return err
	}
	return nil
}

func (q *qp) Listen(host string, port int) error {
	if QLog {
		q.quicConf.Tracer = qlog.NewTracer()
	}
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyOut := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certOut := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	cert, err := tls.X509KeyPair(certOut, keyOut)
	if err != nil {
		log.Fatal(err)
	}

	q.tlsConf = &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"quics-protocol"},
	}

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	if err != nil {
		log.Panicln(err)
		return err
	}
	// tr := quic.Transport{
	// 	Conn: udpConn,
	// }
	// q.quicListener, err = tr.Listen(q.tlsConf, q.quicConf)
	q.quicListener, err = quic.Listen(udpConn, q.tlsConf, q.quicConf)
	if err != nil {
		log.Panicln(err)
		return err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := q.quicListener.Accept(q.ctx)
			if err != nil {
				if err.Error() == ConnectionClosedByPeer {
					log.Println("Connection closed by peer")
					return
				}
				if err.Error() == "quic: server closed" {
					log.Println("Server closed")
					return
				}
				if err.Error() == context.Canceled.Error() {
					log.Println("Context canceled")
					return
				}
				log.Panicln(err)
				return
			}
			log.Println("conn accepted")
			go func(c quic.Connection) {
				str, err := c.AcceptStream(q.ctx)
				if err != nil {
					if err.Error() == ConnectionClosedByPeer {
						log.Println("Connection closed by peer")
						return
					}
					log.Panicln(err)
					return
				}
				defer str.Close()
				log.Println("stream accepted")

				headerSizeBuf := make([]byte, 2)
				log.Println("read header size")
				n, err := io.ReadFull(str, headerSizeBuf)
				if err != nil {
					if err.Error() == ConnectionClosedByPeer {
						log.Println("Connection closed by peer")
						return
					}
					log.Panicln(err)
					return
				}
				if n != 2 {
					log.Panicln("header size is not 2 bytes")
					return
				}
				headerSize := uint16(binary.BigEndian.Uint16(headerSizeBuf))
				headerBuf := make([]byte, headerSize)
				log.Println("read header")
				n, err = io.ReadFull(str, headerBuf)
				if err != nil {
					if err.Error() == ConnectionClosedByPeer {
						log.Println("Connection closed by peer")
						return
					}
					log.Panicln(err)
					return
				}
				if n != int(headerSize) {
					log.Panicln("header size is not ", headerSize, " bytes")
					return
				}
				header := &pb.Header{}
				proto.Unmarshal(headerBuf, header)
				log.Println(header.Type, header.DataSize)

				switch header.Type {
				case pb.MessageType_MESSAGE:
					log.Println("read message")
					messageBuf := make([]byte, header.DataSize)
					n, err = io.ReadFull(str, messageBuf)
					if err != nil {
						if err.Error() == ConnectionClosedByPeer {
							log.Println("Connection closed by peer")
							return
						}
						log.Panicln(err)
						return
					}
					if n != int(header.DataSize) {
						log.Panicln("message size is not ", header.DataSize, " bytes")
						return
					}
					message := &pb.Message{}
					proto.Unmarshal(messageBuf, message)
					log.Println(message.Type, string(message.Message))
					if q.receiveMessageHandler != nil {
						q.receiveMessageHandler(conn, message)
					} else {
						log.Panicln("receiveMessageHandler is nil. Please call RecvMessage() and set handler function for received message before Listen()")
					}
				case pb.MessageType_FILE_INFO:
					log.Println("read file info")
					fileInfoBuf := make([]byte, header.DataSize)
					n, err = io.ReadFull(str, fileInfoBuf)
					if err != nil {
						if err.Error() == ConnectionClosedByPeer {
							log.Println("Connection closed by peer")
							return
						}
						log.Panicln(err)
						return
					}
					if n != int(header.DataSize) {
						log.Panicln("file info size is not ", header.DataSize, " bytes")
						return
					}
					fileInfo := &pb.FileInfo{}
					proto.Unmarshal(fileInfoBuf, fileInfo)
					log.Println(fileInfo.Path, fileInfo.Size, "bytes")
					log.Println("read file")
					fileBuf := make([]byte, fileInfo.Size)
					n, err := io.ReadFull(str, fileBuf)
					if err != nil {
						if err.Error() == ConnectionClosedByPeer {
							log.Println("Connection closed by peer")
							return
						}
						log.Panicln(err)
						return
					}
					if n != int(fileInfo.Size) {
						log.Panicln("file size is not ", fileInfo.Size, " bytes")
						return
					}
					log.Println("file received")
					if q.receiveFileHandler != nil {
						q.receiveFileHandler(conn, fileInfo, fileBuf)
					} else {
						log.Panicln("receiveFileHandler is nil. Please call RecvFile() and set handler function for received file before Listen()")
					}
				default:
					log.Println("unknown message type")
				}
			}(conn)
		}
	}()
	wg.Wait()
	return nil
}

func (q *qp) Close() error {
	if q.quicStream != nil {
		log.Println("Close quicStream")
		err := q.quicStream.Close()
		if err != nil {
			log.Panicln(err)
			return err
		}
	}
	if q.quicListener != nil {
		log.Println("Close quicListener")
		err := q.quicListener.Close()
		if err != nil {
			log.Panicln(err)
			return err
		}
	}
	if q.quicConn != nil {
		log.Println("Close quicConn")
		err := q.quicConn.CloseWithError(0, "Connection closed by peer")
		if err != nil {
			log.Panicln(err)
			return err
		}
	}
	q.cancel()
	return nil
}

func (q *qp) SendFile(path string, file []byte) error {
	fileInfo := &pb.FileInfo{
		Path: path,
		Size: uint64(len(file)),
	}
	fileInfoOut, err := proto.Marshal(fileInfo)
	if err != nil {
		log.Panicln(err)
		return err
	}
	header := &pb.Header{
		Type:     pb.MessageType_FILE_INFO,
		DataSize: uint64(len(fileInfoOut)),
	}
	headerOut, err := proto.Marshal(header)
	if err != nil {
		log.Panicln(err)
		return err
	}

	buf := make([]byte, 2, 2+len(headerOut)+len(fileInfoOut)+len(file))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(headerOut)))
	buf = append(buf, headerOut...)
	buf = append(buf, fileInfoOut...)
	buf = append(buf, file...)

	log.Println("sending ", cap(buf), "bytes")

	n, err := q.quicStream.Write(buf)
	if err != nil {
		log.Panicln(err)
		return err
	}
	log.Println("sent", n)
	return nil
}

func (q *qp) RecvFile(handler func(conn quic.Connection, fileInfo *pb.FileInfo, fileBuf []byte)) error {
	q.receiveFileHandler = handler
	return nil
}

func (q *qp) SendMessage(msgType string, msg []byte) error {
	message := &pb.Message{
		Type:    msgType,
		Message: msg,
	}
	messageOut, err := proto.Marshal(message)
	if err != nil {
		log.Panicln(err)
		return err
	}
	header := &pb.Header{
		Type:     pb.MessageType_MESSAGE,
		DataSize: uint64(len(messageOut)),
	}
	headerOut, err := proto.Marshal(header)
	if err != nil {
		log.Panicln(err)
		return err
	}

	buf := make([]byte, 2, 2+len(headerOut)+len(messageOut))
	binary.BigEndian.PutUint16(buf[:2], uint16(len(headerOut)))
	buf = append(buf, headerOut...)
	buf = append(buf, messageOut...)

	log.Println("sending ", cap(buf), "bytes")

	n, err := q.quicStream.Write(buf)
	if err != nil {
		log.Panicln(err)
		return err
	}
	log.Println("sent", n)
	return nil
}

func (q *qp) RecvMessage(handler func(conn quic.Connection, message *pb.Message)) error {
	q.receiveMessageHandler = handler
	return nil
}
