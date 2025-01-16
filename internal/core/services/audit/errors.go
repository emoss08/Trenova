package audit

import "github.com/rotisserie/eris"

var (
	ErrBufferSizeNotSet    = eris.New("buffer size is not set")
	ErrFlushIntervalNotSet = eris.New("flush interval is not set")
	ErrEmptyBuffer         = eris.New("buffer is empty")
	ErrTimeoutWaitingStop  = eris.New("timeout waiting for audit service to stop")
	ErrServiceStopped      = eris.New("audit service is stopped")
)
