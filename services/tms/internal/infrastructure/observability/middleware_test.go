package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewMiddleware(t *testing.T) {
	t.Parallel()

	tp := &TracerProvider{}
	logger := zap.NewNop()

	m := NewMiddleware(tp, nil, logger)

	assert.NotNil(t, m)
	assert.Same(t, tp, m.tracer)
	assert.Same(t, logger, m.logger)
}
