package tender

import (
	"github.com/shopspring/decimal"
)

// MaxEscalationRound is the most rounds an exhausted tender may be followed
// by, whatever an organization configures.
const MaxEscalationRound = int16(5)

var oneHundred = decimal.NewFromInt(100)

// EscalationRate is an offer's rate raised for an escalation round: up by the
// configured percentage, never above the ceiling the margin floor allows and
// never below what the carrier was already offered. Without a ceiling the
// sell price is unknown, so the rate is not raised at all rather than raised
// blind.
func EscalationRate(
	rate, increasePercent decimal.Decimal,
	ceiling decimal.NullDecimal,
) decimal.Decimal {
	if !ceiling.Valid || !increasePercent.IsPositive() {
		return rate
	}
	raised := rate.Mul(oneHundred.Add(increasePercent)).Div(oneHundred).Round(2)
	capped := ceiling.Decimal.RoundFloor(2)
	if raised.GreaterThan(capped) {
		raised = capped
	}
	if raised.LessThan(rate) {
		return rate
	}
	return raised
}

// RateCeiling is the highest rate that still leaves the margin floor on a
// load that sells for sell. A per-mile rate is the ceiling spread over the
// move's distance; with no sell price, or a per-mile rate and no distance,
// there is no ceiling to give.
func RateCeiling(
	sell, floorPercent decimal.Decimal,
	perMile bool,
	distance *float64,
) decimal.NullDecimal {
	if !sell.IsPositive() {
		return decimal.NullDecimal{}
	}
	maxBuy := sell.Mul(oneHundred.Sub(floorPercent)).Div(oneHundred)
	if !perMile {
		return decimal.NewNullDecimal(maxBuy)
	}
	if distance == nil || *distance <= 0 {
		return decimal.NullDecimal{}
	}
	return decimal.NewNullDecimal(maxBuy.Div(decimal.NewFromFloat(*distance)))
}

// EscalationPlan is the carriers an exhausted tender goes back to: one offer
// per carrier, in the order it was first asked, leaving out any carrier that
// walked an acceptance back. The offers are new rows carrying the old
// delivery details; their rate and expiry are the escalation's to set.
func (t *Tender) EscalationPlan() []*TenderOffer {
	if t == nil {
		return nil
	}
	walkedBack := make(map[string]struct{}, len(t.Offers))
	for _, offer := range t.Offers {
		if offer != nil && offer.Status == OfferStatusRevoked {
			walkedBack[offer.CarrierID.String()] = struct{}{}
		}
	}

	plan := make([]*TenderOffer, 0, len(t.Offers))
	seen := make(map[string]struct{}, len(t.Offers))
	rank := int16(0)
	for _, offer := range t.Offers {
		if offer == nil {
			continue
		}
		key := offer.CarrierID.String()
		if _, out := walkedBack[key]; out {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		next := offer.reoffer()
		next.ReofferedFromID = nil
		next.TenderID = ""
		rank++
		next.Rank = rank
		plan = append(plan, next)
	}
	return plan
}
