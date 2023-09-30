package qp

import (
	qpConn "github.com/quic-s/quics-protocol/pkg/connection"
	qpErr "github.com/quic-s/quics-protocol/pkg/error"
	qpLog "github.com/quic-s/quics-protocol/pkg/log"
	qpStream "github.com/quic-s/quics-protocol/pkg/stream"
	"github.com/quic-s/quics-protocol/pkg/tls"
	"github.com/quic-s/quics-protocol/pkg/types/fileinfo"
)

const (
	LOG_LEVEL_DEBUG = qpLog.DEBUG
	LOG_LEVEL_INFO  = qpLog.INFO
	LOG_LEVEL_ERROR = qpLog.ERROR

	ConnectionClosedByPeer = qpErr.ConnectionClosedByPeer

	NoRecentActivity = qpErr.NoRecentActivity
)

var (
	GetCertificate = tls.GetCertificate
)

type Connection = qpConn.Connection

type Stream = qpStream.Stream

type FileInfo = fileinfo.FileInfo
