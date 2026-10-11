package temporaljobs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracingOptions_TraceTheWorkNotTheStream(t *testing.T) {
	t.Parallel()

	options := tracingOptions()

	assert.True(t, options.DisableSignalTracing, "a stream publish is a signal")
	assert.True(t, options.DisableUpdateTracing, "a stream read is an update")
	assert.True(t, options.DisableQueryTracing)
	assert.False(t, options.DisableBaggage, "baggage still travels with the work")
	assert.Nil(t, options.Tracer, "the interceptor uses the process's tracer provider")
}
