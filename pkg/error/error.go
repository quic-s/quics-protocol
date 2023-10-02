package error

import "errors"

const (
	ConnectionClosedByPeer = "Application error 0x0 (remote): Connection closed by peer"

	NoRecentActivity = "timeout: no recent network activity"

	FileModifiedDuringTransferCode = 0x1
)

var (
	ErrFileModifiedDuringTransfer = errors.New("file modified during transfer")
)
