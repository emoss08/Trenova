package dispatchcost

import "github.com/emoss08/trenova/internal/modules/operations/app/dispatchurgency"

const percent = 100

type MoveValue struct {
	RevenueCents int64
	Urgency      dispatchurgency.Bucket
}

func (m *Model) CoveragePrice(move MoveValue) int64 {
	e := m.economics

	price := max(move.RevenueCents*e.CoverageMarginPercent/percent, e.CoverageFloorCents)
	switch move.Urgency {
	case dispatchurgency.Late:
		price += e.LatePremiumCents
	case dispatchurgency.Now:
		price += e.DueNowPremiumCents
	case dispatchurgency.Today, dispatchurgency.Tomorrow, dispatchurgency.Planned:
	}

	return price
}
