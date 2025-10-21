package audit

import "errors"

var (
	ErrBufferSizeNotSet         = errors.New("buffer size is not set")
	ErrFlushIntervalNotSet      = errors.New("flush interval is not set")
	ErrInvalidConfig            = errors.New("invalid audit configuration")
	ErrEmptyBuffer              = errors.New("buffer is empty")
	ErrTimeoutWaitingStop       = errors.New("timeout waiting for audit service to stop")
	ErrServiceStopped           = errors.New("audit service is stopped")
	ErrServiceNotStarted        = errors.New("audit service is not started")
	ErrInvalidEntry             = errors.New("invalid audit entry")
	ErrSanitizationFailed       = errors.New("failed to sanitize sensitive data")
	ErrRepositoryFailure        = errors.New("audit repository operation failed")
	ErrMaxRetriesExceeded       = errors.New("max retries exceeded for repository operation")
	ErrQueueFull                = errors.New("audit queue is full")
	ErrQueueTimeout             = errors.New("timeout while enqueuing audit entry")
	ErrCompiledPatternNotRegexp = errors.New("compiled pattern is not a regexp")
)
