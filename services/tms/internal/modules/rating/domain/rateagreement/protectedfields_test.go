package rateagreement

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func percent(value string) decimal.NullDecimal {
	return decimal.NewNullDecimal(decimal.RequireFromString(value))
}

func TestAgreementKeepsStoredMarginsForAWriterWithoutTheGroup(t *testing.T) {
	t.Parallel()

	stored := &RateAgreement{MarginFloorPercent: percent("12.5"), MaxPayPercentOfSell: percent("85")}
	incoming := &RateAgreement{Name: "Renewed", MarginFloorPercent: percent("1")}

	incoming.KeepFields(stored, 0)

	assert.Equal(t, "Renewed", incoming.Name)
	assert.True(t, incoming.MarginFloorPercent.Decimal.Equal(decimal.RequireFromString("12.5")))
	assert.True(t, incoming.MaxPayPercentOfSell.Decimal.Equal(decimal.RequireFromString("85")))
}

func TestAgreementTakesMarginsFromAWriterWithTheGroup(t *testing.T) {
	t.Parallel()

	stored := &RateAgreement{MarginFloorPercent: percent("12.5")}
	incoming := &RateAgreement{}

	incoming.KeepFields(stored, permission.AllFieldGroups())

	assert.False(t, incoming.MarginFloorPercent.Valid)
}

func TestAgreementVersionWithholdsMarginsAndTheirChanges(t *testing.T) {
	t.Parallel()

	version := &RateAgreementVersion{
		MarginFloorPercent:  percent("12.5"),
		MaxPayPercentOfSell: percent("85"),
		ChangeSummary: map[string]jsonutils.FieldChange{
			"marginFloorPercent":  {},
			"maxPayPercentOfSell": {},
			"name":                {},
		},
		Agreement: &RateAgreement{MarginFloorPercent: percent("12.5")},
	}

	version.WithholdFields(0)

	assert.False(t, version.MarginFloorPercent.Valid)
	assert.False(t, version.MaxPayPercentOfSell.Valid)
	assert.Equal(t, []string{"name"}, keys(version.ChangeSummary))
	assert.False(t, version.Agreement.MarginFloorPercent.Valid)
}

func keys(m map[string]jsonutils.FieldChange) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	return out
}
