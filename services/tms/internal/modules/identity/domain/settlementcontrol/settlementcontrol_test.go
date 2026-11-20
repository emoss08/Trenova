package settlementcontrol

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefault(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	assert.Equal(t, &SettlementControl{
		OrganizationID:                orgID,
		BusinessUnitID:                buID,
		PayPeriodFrequency:            PayPeriodFrequencyWeekly,
		PeriodEndDayOfWeek:            6,
		PayDelayDays:                  5,
		PayTrigger:                    PayTriggerShipmentDelivered,
		AllowNegativeNet:              true,
		AutoAttachAccruals:            true,
		VarianceThresholdPct:          decimal.NewFromInt(25),
		VarianceLookbackWeeks:         8,
		EscrowInterestFrequencyMonths: 3,
		RequireSeparateApprover:       true,
	}, NewDefault(orgID, buID))
}

func TestSettlementControl_ExplicitZerosSurvive(t *testing.T) {
	t.Parallel()

	sc := NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	sc.PeriodEndDayOfWeek = 0
	sc.PayDelayDays = 0
	sc.VarianceThresholdPct = decimal.Zero

	multiErr := errortypes.NewMultiError()
	sc.Validate(multiErr)
	require.False(t, multiErr.HasErrors())

	_, hooked := any(sc).(interface{ BeforeCreate(context.Context) error })
	assert.False(t, hooked)
	assert.Equal(t, 0, sc.PeriodEndDayOfWeek)
	assert.Equal(t, 0, sc.PayDelayDays)
	assert.True(t, sc.VarianceThresholdPct.IsZero())
}

func TestSettlementControl_ValidateVarianceThresholdBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"zero turns the check off", "0", false},
		{"the column maximum passes", "999.9999", false},
		{"above the column maximum fails", "1000", true},
		{"a fifth decimal that rounds past the maximum fails", "999.99999", true},
		{"negative fails", "-0.0001", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sc := NewDefault(pulid.MustNew("org_"), pulid.MustNew("bu_"))
			sc.VarianceThresholdPct = decimal.RequireFromString(tt.value)

			multiErr := errortypes.NewMultiError()
			sc.Validate(multiErr)

			if !tt.wantErr {
				assert.False(t, multiErr.HasErrors())
				return
			}
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "varianceThresholdPct", multiErr.Errors[0].Field())
			assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
			assert.Equal(
				t,
				"Variance threshold must be between 0 and 999.9999",
				multiErr.Errors[0].Message,
			)
		})
	}
}
