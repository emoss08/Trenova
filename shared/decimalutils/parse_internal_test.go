package decimalutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMoneyText(t *testing.T) {
	t.Parallel()

	value, err := ParseMoneyText("$1,250.50")
	require.NoError(t, err)
	assert.True(t, value.Valid)
	assert.Equal(t, "1250.5", value.Decimal.String())

	empty, err := ParseMoneyText(" ")
	require.NoError(t, err)
	assert.False(t, empty.Valid)

	_, err = ParseMoneyText("call for rate")
	assert.Error(t, err)
}
