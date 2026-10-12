package detention

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestHourlyRate(t *testing.T) {
	t.Parallel()

	assert.True(
		t,
		HourlyRate(decimal.NewFromInt(75), TierRateUnitHour).Equal(decimal.NewFromInt(75)),
	)
	assert.True(
		t,
		HourlyRate(decimal.NewFromInt(480), TierRateUnitDay).Equal(decimal.NewFromInt(20)),
	)
	assert.True(t, HourlyRate(decimal.NewFromInt(150), TierRateUnitFlat).IsZero())
}

func TestPolicySnapshotHourlyRateAt(t *testing.T) {
	t.Parallel()

	second := int32(120)
	snapshot := &PolicySnapshot{
		FlatRate:     decimal.NewFromInt(50),
		FlatRateUnit: TierRateUnitHour,
		Tiers: []TierSnapshot{
			{
				FromMinute: 0,
				ToMinute:   &second,
				Rate:       decimal.NewFromInt(60),
				RateUnit:   TierRateUnitHour,
			},
			{FromMinute: 120, Rate: decimal.NewFromInt(2400), RateUnit: TierRateUnitDay},
		},
	}

	assert.True(t, snapshot.HourlyRateAt(30).Equal(decimal.NewFromInt(60)))
	assert.True(t, snapshot.HourlyRateAt(200).Equal(decimal.NewFromInt(100)))

	snapshot.Tiers = nil
	assert.True(t, snapshot.HourlyRateAt(10).Equal(decimal.NewFromInt(50)))

	var missing *PolicySnapshot
	assert.True(t, missing.HourlyRateAt(10).IsZero())
}
