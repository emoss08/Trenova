package gqlctx

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResponseStatus_RoundTrip(t *testing.T) {
	t.Parallel()

	status := NewResponseStatus()
	ctx := WithResponseStatus(context.Background(), status)

	got, ok := ResponseStatusFrom(ctx)
	assert.True(t, ok)
	assert.Same(t, status, got)

	_, ok = got.Code()
	assert.False(t, ok)
	assert.Zero(t, got.RetryAfter())

	got.Override(http.StatusTooManyRequests, 3*time.Second)

	code, ok := got.Code()
	assert.True(t, ok)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assert.Equal(t, 3*time.Second, got.RetryAfter())
}

func TestResponseStatus_AbsentFromContext(t *testing.T) {
	t.Parallel()

	_, ok := ResponseStatusFrom(context.Background())
	assert.False(t, ok)

	var status *ResponseStatus
	status.Override(http.StatusTooManyRequests, time.Second)
	_, ok = status.Code()
	assert.False(t, ok)
	assert.Zero(t, status.RetryAfter())
}
