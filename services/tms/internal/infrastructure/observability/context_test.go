package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestGetTraceID_NoSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.Equal(t, "", GetTraceID(ctx))
}

func TestGetTraceID_FromContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), TraceIDKey, "abc123")
	assert.Equal(t, "abc123", GetTraceID(ctx))
}

func TestGetSpanID_NoSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.Equal(t, "", GetSpanID(ctx))
}

func TestGetSpanID_FromContext(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), SpanIDKey, "span456")
	assert.Equal(t, "span456", GetSpanID(ctx))
}

func TestWithUserID(t *testing.T) {
	t.Parallel()
	ctx := WithUserID(t.Context(), "user_123")
	userID, ok := ctx.Value(UserIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "user_123", userID)
}

func TestWithOrganizationID(t *testing.T) {
	t.Parallel()
	ctx := WithOrganizationID(t.Context(), "org_456")
	orgID, ok := ctx.Value(OrganizationIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "org_456", orgID)
}

func TestWithAPIKeyID(t *testing.T) {
	t.Parallel()
	ctx := WithAPIKeyID(t.Context(), "key_789")
	apiKeyID, ok := ctx.Value(APIKeyIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "key_789", apiKeyID)
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()
	ctx := WithRequestID(t.Context(), "req_abc")
	requestID, ok := ctx.Value(RequestIDKey).(string)
	assert.True(t, ok)
	assert.Equal(t, "req_abc", requestID)
}

func TestGetUserID_Found(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), UserIDKey, "user_123")
	userID, ok := GetUserID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "user_123", userID)
}

func TestGetUserID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	userID, ok := GetUserID(ctx)
	assert.False(t, ok)
	assert.Equal(t, "", userID)
}

func TestGetOrganizationID_Found(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), OrganizationIDKey, "org_456")
	orgID, ok := GetOrganizationID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "org_456", orgID)
}

func TestGetOrganizationID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	orgID, ok := GetOrganizationID(ctx)
	assert.False(t, ok)
	assert.Equal(t, "", orgID)
}

func TestGetAPIKeyID_Found(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), APIKeyIDKey, "key_789")
	keyID, ok := GetAPIKeyID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "key_789", keyID)
}

func TestGetAPIKeyID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	keyID, ok := GetAPIKeyID(ctx)
	assert.False(t, ok)
	assert.Equal(t, "", keyID)
}

func TestGetRequestID_Found(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), RequestIDKey, "req_abc")
	reqID, ok := GetRequestID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "req_abc", reqID)
}

func TestGetRequestID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	reqID, ok := GetRequestID(ctx)
	assert.False(t, ok)
	assert.Equal(t, "", reqID)
}

func TestWithTraceIDs_NoSpan(t *testing.T) {
	t.Parallel()
	ctx := WithTraceIDs(t.Context())
	assert.NotNil(t, ctx)
}

func TestSpanFromContext_NoSpan(t *testing.T) {
	t.Parallel()
	span := SpanFromContext(t.Context())
	assert.NotNil(t, span)
}

func TestAddSpanAttributes_NoSpan(t *testing.T) {
	t.Parallel()
	AddSpanAttributes(t.Context())
}

func TestAddSpanEvent_NoSpan(t *testing.T) {
	t.Parallel()
	AddSpanEvent(t.Context(), "test-event")
}

func TestRecordSpanError_NoSpan(t *testing.T) {
	t.Parallel()
	RecordSpanError(t.Context(), errors.New("test error"))
}

func TestRecordSpanError_NilError(t *testing.T) {
	t.Parallel()
	RecordSpanError(t.Context(), nil)
}

func TestSetSpanOK_NoSpan(t *testing.T) {
	t.Parallel()
	SetSpanOK(t.Context(), "ok")
}

func TestStartSpanFromContext(t *testing.T) {
	t.Parallel()
	ctx, span := StartSpanFromContext(t.Context(), "test-span")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func TestRunWithSpan_Success(t *testing.T) {
	t.Parallel()
	called := false
	err := RunWithSpan(t.Context(), "test-op", func(_ context.Context) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRunWithSpan_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("operation failed")
	err := RunWithSpan(t.Context(), "test-op", func(_ context.Context) error {
		return expectedErr
	})
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestRunWithSpanReturn_Success(t *testing.T) {
	t.Parallel()
	result, err := RunWithSpanReturn(
		t.Context(),
		"test-op",
		func(_ context.Context) (int, error) {
			return 42, nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestRunWithSpanReturn_Error(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("operation failed")
	result, err := RunWithSpanReturn(
		t.Context(),
		"test-op",
		func(_ context.Context) (int, error) {
			return 0, expectedErr
		},
	)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 0, result)
}

func TestExtractTraceState_Empty(t *testing.T) {
	t.Parallel()
	state := ExtractTraceState(t.Context())
	assert.Empty(t, state)
}

func TestExtractTraceState_WithValues(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	ctx = context.WithValue(ctx, UserIDKey, "user_123")
	ctx = context.WithValue(ctx, OrganizationIDKey, "org_456")
	ctx = context.WithValue(ctx, RequestIDKey, "req_789")

	state := ExtractTraceState(ctx)
	assert.Equal(t, "user_123", state["user_id"])
	assert.Equal(t, "org_456", state["organization_id"])
	assert.Equal(t, "req_789", state["request_id"])
}

func TestGetUserID_FromBaggage(t *testing.T) {
	t.Parallel()
	ctx := WithUserID(t.Context(), "user_baggage")
	userID, ok := GetUserID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "user_baggage", userID)
}

func TestGetOrganizationID_FromBaggage(t *testing.T) {
	t.Parallel()
	ctx := WithOrganizationID(t.Context(), "org_baggage")
	orgID, ok := GetOrganizationID(ctx)
	assert.True(t, ok)
	assert.Equal(t, "org_baggage", orgID)
}

func TestAddSpanAttributes_NoActiveSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.NotPanics(t, func() {
		AddSpanAttributes(ctx, attribute.String("key", "value"), attribute.Int("num", 42))
	})
}

func TestAddSpanEvent_NoActiveSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.NotPanics(t, func() {
		AddSpanEvent(ctx, "my-event", attribute.String("detail", "info"))
	})
}

func TestRecordSpanError_NilErrorNoSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.NotPanics(t, func() {
		RecordSpanError(ctx, nil)
	})
}

func TestRecordSpanError_WithErrorNoSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.NotPanics(t, func() {
		RecordSpanError(ctx, errors.New("some failure"))
	})
}

func TestSetSpanOK_NoActiveSpan(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	assert.NotPanics(t, func() {
		SetSpanOK(ctx, "all good")
	})
}

func TestGetTraceID_NoSpanNoContext(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	result := GetTraceID(ctx)
	assert.Equal(t, "", result)
}

func TestGetSpanID_NoSpanNoContext(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	result := GetSpanID(ctx)
	assert.Equal(t, "", result)
}

func TestWithTraceIDs_NoActiveSpan(t *testing.T) {
	t.Parallel()
	ctx := WithTraceIDs(t.Context())
	traceID, _ := ctx.Value(TraceIDKey).(string)
	spanID, _ := ctx.Value(SpanIDKey).(string)
	assert.Equal(t, "", traceID)
	assert.Equal(t, "", spanID)
}

func TestWithBaggageMember_InvalidKey(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	result := withBaggageMember(ctx, string([]byte{0x00}), "value")
	assert.Equal(t, ctx, result)
}

func TestExtractTraceState_WithSpanID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), SpanIDKey, "span_abc")
	state := ExtractTraceState(ctx)
	assert.Equal(t, "span_abc", state["span_id"])
}

func TestExtractTraceState_WithTraceID(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(t.Context(), TraceIDKey, "trace_xyz")
	state := ExtractTraceState(ctx)
	assert.Equal(t, "trace_xyz", state["trace_id"])
}

func TestSpanFromContext_ReturnsNonRecordingSpan(t *testing.T) {
	t.Parallel()
	span := SpanFromContext(t.Context())
	assert.False(t, span.IsRecording())
}
