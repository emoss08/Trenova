package detention_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func validPolicy() *detention.DetentionPolicy {
	return &detention.DetentionPolicy{
		Name:                    "Standard",
		Code:                    "STD",
		Status:                  detention.PolicyStatusActive,
		ClockStartBasis:         detention.ClockStartBasisArrival,
		LateArrivalRule:         detention.LateArrivalRuleNoEffect,
		RoundingMode:            detention.RoundingModeUp,
		BillingIncrementMinutes: 15,
		BillingFreeMinutes:      120,
		RateSource:              detention.RateSourceAccessorial,
		AccessorialChargeID:     pulid.MustNew("acc_"),
		NotificationRequirement: detention.NotificationRequirementNone,
		UnnotifiedBehavior:      detention.UnnotifiedBehaviorBill,
		DayBoundaryMode:         detention.CapScopePerStop,
		Currency:                "USD",
	}
}

func validate(p *detention.DetentionPolicy) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)
	return multiErr
}

func errorFields(multiErr *errortypes.MultiError) []string {
	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}
	return fields
}

func TestPolicy_ValidBaseline(t *testing.T) {
	t.Parallel()

	assert.False(t, validate(validPolicy()).HasErrors())
}

func TestPolicy_SpecificityScoring(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*detention.DetentionPolicy)
		want   int32
	}{
		{"unscoped policy scores zero", func(*detention.DetentionPolicy) {}, 0},
		{
			"location only",
			func(p *detention.DetentionPolicy) { p.LocationID = ptr(pulid.MustNew("loc_")) },
			32,
		},
		{
			"customer only",
			func(p *detention.DetentionPolicy) { p.CustomerID = ptr(pulid.MustNew("cus_")) },
			16,
		},
		{
			"customer plus location outranks either alone",
			func(p *detention.DetentionPolicy) {
				p.CustomerID = ptr(pulid.MustNew("cus_"))
				p.LocationID = ptr(pulid.MustNew("loc_"))
			},
			48,
		},
		{
			"every dimension set",
			func(p *detention.DetentionPolicy) {
				p.LocationID = ptr(pulid.MustNew("loc_"))
				p.CustomerID = ptr(pulid.MustNew("cus_"))
				p.CommodityIDs = []pulid.ID{pulid.MustNew("cmd_")}
				p.ServiceTypeIDs = []pulid.ID{pulid.MustNew("svt_")}
				p.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("spt_")}
				p.StopTypes = []shipment.StopType{shipment.StopTypeDelivery}
			},
			63,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := validPolicy()
			tt.mutate(p)

			assert.Equal(t, tt.want, p.ComputeSpecificity())
		})
	}
}

func TestPolicy_Matches(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	otherCustomerID := pulid.MustNew("cus_")
	locationID := pulid.MustNew("loc_")
	commodityID := pulid.MustNew("cmd_")

	base := detention.MatchInput{
		CustomerID:     customerID,
		LocationID:     locationID,
		ShipmentTypeID: pulid.MustNew("spt_"),
		ServiceTypeID:  pulid.MustNew("svt_"),
		CommodityIDs:   []pulid.ID{commodityID},
		StopType:       shipment.StopTypeDelivery,
		ScheduleType:   shipment.StopScheduleTypeAppointment,
		Timestamp:      1000,
	}

	t.Run("wildcard policy matches everything", func(t *testing.T) {
		t.Parallel()
		assert.True(t, validPolicy().Matches(base))
	})

	t.Run("customer scope matches its customer", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.CustomerID = &customerID
		assert.True(t, p.Matches(base))
	})

	t.Run("customer scope rejects a different customer", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.CustomerID = &otherCustomerID
		assert.False(t, p.Matches(base))
	})

	t.Run("inactive policy never matches", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.Status = detention.PolicyStatusInactive
		assert.False(t, p.Matches(base))
	})

	t.Run("draft policy never matches", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.Status = detention.PolicyStatusDraft
		assert.False(t, p.Matches(base))
	})

	t.Run("commodity scope matches on any overlap", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.CommodityIDs = []pulid.ID{pulid.MustNew("cmd_"), commodityID}
		assert.True(t, p.Matches(base))
	})

	t.Run("commodity scope rejects a disjoint set", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.CommodityIDs = []pulid.ID{pulid.MustNew("cmd_")}
		assert.False(t, p.Matches(base))
	})

	t.Run("stop type scope filters by leg", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.StopTypes = []shipment.StopType{shipment.StopTypePickup}
		assert.False(t, p.Matches(base))

		p.StopTypes = []shipment.StopType{shipment.StopTypePickup, shipment.StopTypeDelivery}
		assert.True(t, p.Matches(base))
	})

	t.Run("appointment-only policy skips open stops", func(t *testing.T) {
		t.Parallel()
		p := validPolicy()
		p.AppointmentStopsOnly = true

		open := base
		open.ScheduleType = shipment.StopScheduleTypeOpen

		assert.False(t, p.Matches(open))
		assert.True(t, p.Matches(base))
	})
}

func TestPolicy_EffectiveDating(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	p.EffectiveStartDate = ptr(int64(1000))
	p.EffectiveEndDate = ptr(int64(2000))

	assert.False(t, p.IsEffectiveAt(999), "before the window")
	assert.True(t, p.IsEffectiveAt(1000), "the start boundary is inclusive")
	assert.True(t, p.IsEffectiveAt(1500))
	assert.True(t, p.IsEffectiveAt(2000), "the end boundary is inclusive")
	assert.False(t, p.IsEffectiveAt(2001), "after the window")
}

func TestPolicy_FreeMinutesForStopType(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	p.BillingFreeMinutes = 120
	p.PickupFreeMinutes = ptr(int32(60))
	p.DeliveryFreeMinutes = ptr(int32(180))

	assert.Equal(t, int32(60), p.FreeMinutesForStopType(shipment.StopTypePickup))
	assert.Equal(t, int32(60), p.FreeMinutesForStopType(shipment.StopTypeSplitPickup))
	assert.Equal(t, int32(180), p.FreeMinutesForStopType(shipment.StopTypeDelivery))
	assert.Equal(t, int32(180), p.FreeMinutesForStopType(shipment.StopTypeSplitDelivery))

	p.PickupFreeMinutes = nil
	assert.Equal(t, int32(120), p.FreeMinutesForStopType(shipment.StopTypePickup),
		"an absent override falls back to the general allowance")
}

func TestPolicy_PayFreeMinutesOrDefault(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	assert.Equal(t, int32(120), p.PayFreeMinutesOrDefault())

	p.PayFreeMinutes = ptr(int32(90))
	assert.Equal(t, int32(90), p.PayFreeMinutesOrDefault())
}

func TestPolicy_ValidationRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mutate    func(*detention.DetentionPolicy)
		wantField string
	}{
		{
			name:      "free time above the seven day ceiling",
			mutate:    func(p *detention.DetentionPolicy) { p.BillingFreeMinutes = 20000 },
			wantField: "billingFreeMinutes",
		},
		{
			name:      "negative free time",
			mutate:    func(p *detention.DetentionPolicy) { p.BillingFreeMinutes = -1 },
			wantField: "billingFreeMinutes",
		},
		{
			name:      "zero increment without exact rounding",
			mutate:    func(p *detention.DetentionPolicy) { p.BillingIncrementMinutes = 0 },
			wantField: "billingIncrementMinutes",
		},
		{
			name: "expiration before effective date",
			mutate: func(p *detention.DetentionPolicy) {
				p.EffectiveStartDate = ptr(int64(2000))
				p.EffectiveEndDate = ptr(int64(1000))
			},
			wantField: "effectiveEndDate",
		},
		{
			name: "shipment cap below stop cap",
			mutate: func(p *detention.DetentionPolicy) {
				p.MaxChargePerStop = decimal.NewNullDecimal(decimal.NewFromInt(500))
				p.MaxChargePerShipment = decimal.NewNullDecimal(decimal.NewFromInt(200))
			},
			wantField: "maxChargePerShipment",
		},
		{
			name: "layover threshold without a layover charge",
			mutate: func(p *detention.DetentionPolicy) {
				p.ConvertToLayoverAtMinutes = ptr(int32(1440))
			},
			wantField: "layoverAccessorialChargeId",
		},
		{
			name: "required notice that still bills when missed",
			mutate: func(p *detention.DetentionPolicy) {
				p.NotificationRequirement = detention.NotificationRequirementRequired
				p.UnnotifiedBehavior = detention.UnnotifiedBehaviorBill
			},
			wantField: "unnotifiedBehavior",
		},
		{
			name: "notice lead longer than the free time it warns about",
			mutate: func(p *detention.DetentionPolicy) {
				p.NotificationRequirement = detention.NotificationRequirementAdvisory
				p.BillingFreeMinutes = 60
				p.NotificationLeadMinutes = 90
			},
			wantField: "notificationLeadMinutes",
		},
		{
			name: "tier rate source without tiers",
			mutate: func(p *detention.DetentionPolicy) {
				p.RateSource = detention.RateSourceTiers
			},
			wantField: "tiers",
		},
		{
			name: "org default policy carrying scope",
			mutate: func(p *detention.DetentionPolicy) {
				p.IsOrgDefault = true
				p.CustomerID = ptr(pulid.MustNew("cus_"))
			},
			wantField: "isOrgDefault",
		},
		{
			name: "auto-approve above the approval-required threshold",
			mutate: func(p *detention.DetentionPolicy) {
				p.RequireApprovalOverAmount = decimal.NewNullDecimal(decimal.NewFromInt(100))
				p.AutoApproveUnderAmount = decimal.NewNullDecimal(decimal.NewFromInt(500))
			},
			wantField: "autoApproveUnderAmount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := validPolicy()
			tt.mutate(p)

			multiErr := validate(p)
			require.True(t, multiErr.HasErrors(), "expected a validation failure")
			assert.Contains(t, errorFields(multiErr), tt.wantField)
		})
	}
}

func TestPolicy_ExactRoundingAllowsZeroIncrement(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	p.RoundingMode = detention.RoundingModeExact
	p.BillingIncrementMinutes = 0

	assert.False(t, validate(p).HasErrors())
}

func TestPolicy_RequiresNotice(t *testing.T) {
	t.Parallel()

	p := validPolicy()
	assert.False(t, p.RequiresNotice())

	p.NotificationRequirement = detention.NotificationRequirementAdvisory
	assert.False(t, p.RequiresNotice(), "advisory notices do not gate billing")

	p.NotificationRequirement = detention.NotificationRequirementRequired
	assert.True(t, p.RequiresNotice())
}
