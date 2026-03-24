package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestNewContextLogger(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	assert.NotNil(t, cl)
	assert.Same(t, logger, cl.base)
}

func TestContextLogger_Logger(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	assert.Same(t, logger, cl.Logger())
}

func TestContextLogger_WithContext_Empty(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	result := cl.WithContext(t.Context())
	assert.NotNil(t, result)
}

func TestContextLogger_WithContext_WithValues(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	ctx := t.Context()
	ctx = context.WithValue(ctx, UserIDKey, "user_123")
	ctx = context.WithValue(ctx, OrganizationIDKey, "org_456")
	ctx = context.WithValue(ctx, RequestIDKey, "req_789")

	result := cl.WithContext(ctx)
	assert.NotNil(t, result)
}

func TestContextLogger_WithContext_TraceAndSpan(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	ctx := t.Context()
	ctx = context.WithValue(ctx, TraceIDKey, "trace_abc")
	ctx = context.WithValue(ctx, SpanIDKey, "span_def")

	result := cl.WithContext(ctx)
	assert.NotNil(t, result)
}

func TestContextLogger_Debug(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	cl.Debug(t.Context(), "debug message", zap.String("key", "value"))
}

func TestContextLogger_Info(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	cl.Info(t.Context(), "info message", zap.Int("count", 5))
}

func TestContextLogger_Warn(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	cl.Warn(t.Context(), "warn message")
}

func TestContextLogger_Error(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	cl.Error(t.Context(), "error message", zap.Error(assert.AnError))
}

func TestContextLogger_With(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	derived := cl.With(zap.String("service", "test"))
	assert.NotNil(t, derived)
	assert.NotSame(t, cl, derived)
}

func TestContextLogger_Named(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	named := cl.Named("subsystem")
	assert.NotNil(t, named)
	assert.NotSame(t, cl, named)
}

func TestContextLogger_WithContext_AllFields(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	ctx := t.Context()
	ctx = context.WithValue(ctx, TraceIDKey, "trace_id_value")
	ctx = context.WithValue(ctx, SpanIDKey, "span_id_value")
	ctx = context.WithValue(ctx, UserIDKey, "user_id_value")
	ctx = context.WithValue(ctx, OrganizationIDKey, "org_id_value")
	ctx = context.WithValue(ctx, RequestIDKey, "request_id_value")

	result := cl.WithContext(ctx)
	assert.NotNil(t, result)
	assert.NotSame(t, logger, result)
}

func TestContextLogger_Debug_WithContextValues(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	ctx := context.WithValue(t.Context(), UserIDKey, "user_abc")
	cl.Debug(ctx, "context debug")
}

func TestContextLogger_Info_WithContextValues(t *testing.T) {
	t.Parallel()

	logger := zaptest.NewLogger(t)
	cl := NewContextLogger(logger)

	ctx := context.WithValue(t.Context(), OrganizationIDKey, "org_xyz")
	cl.Info(ctx, "context info")
}
