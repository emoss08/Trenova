// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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

	// Queue errors
	ErrQueueFull    = eris.New("audit queue is full")
	ErrQueueTimeout = eris.New("timeout while enqueuing audit entry")
)
