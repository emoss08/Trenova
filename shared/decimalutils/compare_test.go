package decimalutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMax(t *testing.T) {
	t.Parallel()

	assert.True(t, Max(dec("10.50"), dec("9")).Equal(dec("10.50")))
	assert.True(t, Max(dec("9"), dec("10.50")).Equal(dec("10.50")))
	assert.True(t, Max(dec("-4"), dec("-9")).Equal(dec("-4")))
	assert.True(t, Max(dec("5"), dec("5")).Equal(dec("5")))
}

func TestMin(t *testing.T) {
	t.Parallel()

	assert.True(t, Min(dec("10.50"), dec("9")).Equal(dec("9")))
	assert.True(t, Min(dec("9"), dec("10.50")).Equal(dec("9")))
	assert.True(t, Min(dec("-4"), dec("-9")).Equal(dec("-9")))
}

func TestAbs(t *testing.T) {
	t.Parallel()

	assert.True(t, Abs(dec("12.34")).Equal(dec("12.34")))
	assert.True(t, Abs(dec("-12.34")).Equal(dec("12.34")))
	assert.True(t, Abs(dec("0")).Equal(dec("0")))
}
