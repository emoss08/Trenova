package auditjobs

import "errors"

var (
	ErrBufferFull               = errors.New("buffer is full")
	ErrBufferCircuitBreakerOpen = errors.New("buffer circuit breaker is open")
)
