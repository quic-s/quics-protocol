package log

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/quic-go/quic-go/qlog"
)

type BufferedWriteCloser struct {
	*bufio.Writer
	io.Closer
}

func NewQLogTracer() func(context.Context, logging.Perspective, quic.ConnectionID) *logging.ConnectionTracer {
	return func(ctx context.Context, p logging.Perspective, connID quic.ConnectionID) *logging.ConnectionTracer {
		filename := fmt.Sprintf("client_%x.qlog", connID)
		f, err := os.Create(filename)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Creating qlog file %s.\n", filename)
		return qlog.NewConnectionTracer(newBufferedWriteCloser(bufio.NewWriter(f), f), p, connID)
	}
}

// NewBufferedWriteCloser creates an io.WriteCloser from a bufio.Writer and an io.Closer
func newBufferedWriteCloser(writer *bufio.Writer, closer io.Closer) io.WriteCloser {
	return &BufferedWriteCloser{
		Writer: writer,
		Closer: closer,
	}
}

func (h *BufferedWriteCloser) Close() error {
	if err := h.Writer.Flush(); err != nil {
		return err
	}
	return h.Closer.Close()
}
