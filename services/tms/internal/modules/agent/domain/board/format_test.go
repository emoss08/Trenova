package board_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFormatValueKeepsARateBelowADollarToATenthOfACent(t *testing.T) {
	t.Parallel()

	rate := boardcatalog.Amount(decimal.RequireFromString("0.638"), "USD")
	assert.Equal(t, "$0.638", board.FormatValue(&rate))
	even := boardcatalog.Amount(decimal.RequireFromString("0.640"), "USD")
	assert.Equal(t, "$0.64", board.FormatValue(&even))
	cents := boardcatalog.Money(64, "USD")
	assert.Equal(t, "$0.64", board.FormatValue(&cents))
	whole := boardcatalog.Money(135240, "USD")
	assert.Equal(t, "$1,352.40", board.FormatValue(&whole))
}
