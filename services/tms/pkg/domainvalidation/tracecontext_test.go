package domainvalidation

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTraceID(t *testing.T) {
	t.Parallel()

	assert.True(t, IsTraceID("4bf92f3577b34da6a3ce929d0e0e4736"))
	assert.False(t, IsTraceID("00000000000000000000000000000000"))
	assert.False(t, IsTraceID("4BF92F3577B34DA6A3CE929D0E0E4736"))
	assert.False(t, IsTraceID("4bf92f3577b34da6a3ce929d0e0e473"))
	assert.False(t, IsTraceID(""))
}

func TestIsSpanID(t *testing.T) {
	t.Parallel()

	assert.True(t, IsSpanID("00f067aa0ba902b7"))
	assert.False(t, IsSpanID("0000000000000000"))
	assert.False(t, IsSpanID("00f067aa0ba902b7aa"))
	assert.False(t, IsSpanID("00f067aa0ba902bz"))
}

func TestTraceIDRule(t *testing.T) {
	t.Parallel()

	rule := TraceID("Trace id is invalid")

	require.NoError(t, validation.Validate("", rule))
	require.NoError(t, validation.Validate("4bf92f3577b34da6a3ce929d0e0e4736", rule))
	require.EqualError(t, validation.Validate("not-a-trace", rule), "Trace id is invalid")
	require.EqualError(t, validation.Validate(42, rule), "Trace id is invalid")
}

func TestSpanIDRule(t *testing.T) {
	t.Parallel()

	rule := SpanID("Span id is invalid")

	require.NoError(t, validation.Validate("", rule))
	require.NoError(t, validation.Validate("00f067aa0ba902b7", rule))
	require.EqualError(t, validation.Validate("0000000000000000", rule), "Span id is invalid")
}
