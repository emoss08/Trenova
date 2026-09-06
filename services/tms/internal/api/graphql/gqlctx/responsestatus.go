package gqlctx

import (
	"context"
	"sync"
	"time"
)

type responseStatusKey struct{}

type ResponseStatus struct {
	mu         sync.Mutex
	code       int
	retryAfter time.Duration
}

func NewResponseStatus() *ResponseStatus {
	return &ResponseStatus{}
}

func WithResponseStatus(ctx context.Context, status *ResponseStatus) context.Context {
	return context.WithValue(ctx, responseStatusKey{}, status)
}

func ResponseStatusFrom(ctx context.Context) (*ResponseStatus, bool) {
	status, ok := ctx.Value(responseStatusKey{}).(*ResponseStatus)
	return status, ok && status != nil
}

func (s *ResponseStatus) Override(code int, retryAfter time.Duration) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.code = code
	s.retryAfter = retryAfter
}

func (s *ResponseStatus) Code() (int, bool) {
	if s == nil {
		return 0, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.code, s.code != 0
}

func (s *ResponseStatus) RetryAfter() time.Duration {
	if s == nil {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.retryAfter
}
