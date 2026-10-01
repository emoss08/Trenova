package requestmeta

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithAndFrom(t *testing.T) {
	t.Parallel()

	_, ok := From(t.Context())
	assert.False(t, ok)

	meta := New("req-1", "203.0.113.9", "agent")
	got, ok := From(With(t.Context(), meta))
	assert.True(t, ok)
	assert.Equal(t, meta, got)
}

func TestFromFallsBackToTheGinKey(t *testing.T) {
	t.Parallel()

	meta := New("req-2", "198.51.100.4", "agent")
	ctx := context.WithValue(t.Context(), GinContextKey, meta) //nolint:staticcheck // mirrors gin.Context.Value
	got, ok := From(ctx)
	assert.True(t, ok)
	assert.Equal(t, meta, got)
}

func TestNewTruncatesToTheAuditColumns(t *testing.T) {
	t.Parallel()

	meta := New("r", strings.Repeat("1", 60), strings.Repeat("a", 400))
	assert.Len(t, meta.ClientIP, maxClientIPLength)
	assert.Len(t, meta.UserAgent, maxUserAgentLength)
}
