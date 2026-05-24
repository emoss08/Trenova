package helpers

import (
	"context"
	"fmt"
	"time"
)

type RequestTimeoutError struct {
	Timeout time.Duration
}

func NewRequestTimeoutError(timeout time.Duration) *RequestTimeoutError {
	return &RequestTimeoutError{Timeout: timeout}
}

func (e *RequestTimeoutError) Error() string {
	return fmt.Sprintf("request exceeded app deadline of %s", e.Timeout)
}

func (e *RequestTimeoutError) Is(target error) bool {
	return target == context.DeadlineExceeded
}
