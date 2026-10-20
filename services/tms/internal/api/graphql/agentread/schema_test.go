package agentread

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewSchema_HoldsWhatEveryBoundOperationReaches(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		NewSchema(ResolverParams{Logger: zap.NewNop()})
	})
}
