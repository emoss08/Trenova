package decimalutils

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestPtrEqualComparesByValue(t *testing.T) {
	t.Parallel()

	ten := decimal.RequireFromString("10")
	tenPointZero := decimal.RequireFromString("10.00")
	eleven := decimal.NewFromInt(11)

	assert.True(t, PtrEqual(nil, nil))
	assert.True(t, PtrEqual(&ten, &tenPointZero))
	assert.False(t, PtrEqual(&ten, &eleven))
	assert.False(t, PtrEqual(&ten, nil))
	assert.False(t, PtrEqual(nil, &ten))
}
