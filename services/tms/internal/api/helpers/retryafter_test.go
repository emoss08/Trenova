package helpers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
)

func TestErrorHandler_HandleError_RateLimitSetsRetryAfter(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext().WithPath("/api/v1/auth/login")

	handler.HandleError(
		ctx.Context,
		errortypes.NewRateLimitError("emailAddress", "Too many attempts").
			WithRetryAfter(1500*time.Millisecond),
	)

	assert.Equal(t, http.StatusTooManyRequests, ctx.ResponseCode())
	assert.Equal(t, "2", ctx.ResponseHeader("Retry-After"))
}

func TestErrorHandler_HandleError_RateLimitWithoutRetryAfter(t *testing.T) {
	t.Parallel()

	handler := newTestErrorHandler(false)
	ctx := testutil.NewGinTestContext().WithPath("/api/v1/auth/login")

	handler.HandleError(ctx.Context, errortypes.NewRateLimitError("", "Too many attempts"))

	assert.Equal(t, http.StatusTooManyRequests, ctx.ResponseCode())
	assert.Empty(t, ctx.ResponseHeader("Retry-After"))
}
