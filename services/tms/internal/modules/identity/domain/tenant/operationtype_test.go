package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOperationTypeCoverageNoun(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a driver", OperationTypeAsset.CoverageNoun())
	assert.Equal(t, "a carrier", OperationTypeBrokerage.CoverageNoun())
	assert.Equal(t, "coverage", OperationTypeBoth.CoverageNoun())
	assert.Equal(t, OperationTypeBoth, OperationTypeOf(true, true))
	assert.Equal(t, OperationTypeBrokerage, OperationTypeOf(true, false))
	assert.Equal(t, OperationTypeAsset, OperationTypeOf(false, false))
}
