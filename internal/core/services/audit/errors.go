package audit

import "github.com/rotisserie/eris"

var (
	// Configuration errors
	ErrBufferSizeNotSet    = eris.New("buffer size is not set")
	ErrFlushIntervalNotSet = eris.New("flush interval is not set")
	ErrInvalidConfig       = eris.New("invalid audit configuration")

	// Operation errors
	ErrEmptyBuffer        = eris.New("buffer is empty")
	ErrTimeoutWaitingStop = eris.New("timeout waiting for audit service to stop")
	ErrServiceStopped     = eris.New("audit service is stopped")
	ErrServiceNotStarted  = eris.New("audit service is not started")

	// Data errors
	ErrInvalidEntry       = eris.New("invalid audit entry")
	ErrSanitizationFailed = eris.New("failed to sanitize sensitive data")

	// Storage errors
	ErrRepositoryFailure  = eris.New("audit repository operation failed")
	ErrMaxRetriesExceeded = eris.New("max retries exceeded for repository operation")
)
