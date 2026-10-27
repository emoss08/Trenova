package dispatchcontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAnUpdateWithoutTheMarginsGroupKeepsTheStoredCounterFloor(t *testing.T) {
	t.Parallel()

	stored := &DispatchControl{
		TenderCounterMarginFloorPercent: decimal.NewNullDecimal(decimal.NewFromInt(12)),
	}

	sent := &DispatchControl{}
	sent.KeepFields(stored, 0)
	assert.True(t, decimal.NewFromInt(12).Equal(sent.TenderCounterMarginFloorPercent.Decimal),
		"a form that never received the floor cannot clear it")

	changed := &DispatchControl{
		TenderCounterMarginFloorPercent: decimal.NewNullDecimal(decimal.NewFromInt(4)),
	}
	changed.KeepFields(stored, permission.AllFieldGroups())
	assert.True(t, decimal.NewFromInt(4).Equal(changed.TenderCounterMarginFloorPercent.Decimal),
		"a writer with the margins group sets the floor")
}
