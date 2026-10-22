package costingcontrol

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestAnUpdateWithoutTheMarginsGroupKeepsTheStoredTargetMargin(t *testing.T) {
	t.Parallel()

	stored := &CostingControl{TargetMarginPercent: decimal.NewNullDecimal(decimal.NewFromInt(18))}

	sent := &CostingControl{}
	sent.KeepFields(stored, 0)
	assert.True(t, sent.TargetMarginPercent.Valid)
	assert.True(t, decimal.NewFromInt(18).Equal(sent.TargetMarginPercent.Decimal))

	cleared := &CostingControl{}
	cleared.KeepFields(stored, permission.AllFieldGroups())
	assert.False(t, cleared.TargetMarginPercent.Valid,
		"a writer with the margins group may clear the target")

	shown := &CostingControl{TargetMarginPercent: stored.TargetMarginPercent}
	shown.WithholdFields(0)
	assert.False(t, shown.TargetMarginPercent.Valid)
}
