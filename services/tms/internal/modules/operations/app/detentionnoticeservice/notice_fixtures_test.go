package detentionnoticeservice_test

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	baseTime = int64(1767225600)
	hour     = int64(3600)
)

func ptr[T any](v T) *T { return &v }

func money(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		panic(err)
	}
	return d
}

// baseSnapshot is the industry-default contract: 2 hours free, billed hourly at
// $75, rounded up to 15-minute increments.
func baseSnapshot() detentionpolicy.PolicySnapshot {
	return detentionpolicy.PolicySnapshot{
		PolicyID:                pulid.MustNew("dtp_"),
		PolicyCode:              "STD-2HR",
		PolicyName:              "Standard 2 Hour Free",
		ClockStartBasis:         detentionpolicy.ClockStartBasisArrival,
		LateArrivalRule:         detentionpolicy.LateArrivalRuleNoEffect,
		FreeMinutes:             120,
		PayFreeMinutes:          120,
		BillingIncrementMinutes: 15,
		RoundingMode:            detentionpolicy.RoundingModeUp,
		RateSource:              detentionpolicy.RateSourceAccessorial,
		AccessorialCode:         "DET",
		FlatRate:                money("75.00"),
		FlatRateUnit:            detentionpolicy.TierRateUnitHour,
		DayBoundaryMode:         detentionpolicy.CapScopePerStop,
		NotificationRequirement: detentionpolicy.NotificationRequirementNone,
		UnnotifiedBehavior:      detentionpolicy.UnnotifiedBehaviorBill,
		Currency:                "USD",
	}
}
