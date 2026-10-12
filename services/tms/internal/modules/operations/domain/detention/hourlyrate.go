package detention

import "github.com/shopspring/decimal"

var hoursPerDay = decimal.NewFromInt(24)

func HourlyRate(rate decimal.Decimal, unit TierRateUnit) decimal.Decimal {
	switch unit {
	case TierRateUnitHour:
		return rate
	case TierRateUnitDay:
		return rate.Div(hoursPerDay)
	case TierRateUnitFlat:
		return decimal.Zero
	default:
		return decimal.Zero
	}
}

func (s *PolicySnapshot) HourlyRateAt(billableMinutes int32) decimal.Decimal {
	if s == nil {
		return decimal.Zero
	}

	for i := range s.Tiers {
		tier := &s.Tiers[i]
		if billableMinutes < tier.FromMinute {
			continue
		}
		if tier.ToMinute != nil && billableMinutes >= *tier.ToMinute {
			continue
		}
		return HourlyRate(tier.Rate, tier.RateUnit)
	}

	return HourlyRate(s.FlatRate, s.FlatRateUnit)
}
