package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/shopspring/decimal"
)

const centsPlaces = 2

type PricedAgreementAccessorial struct {
	Accessorial *rateagreement.RateAgreementAccessorial
	Charge      *AdditionalCharge
}

func (sp *Shipment) AccessorialMeasures() *rateagreement.AccessorialMeasures {
	return &rateagreement.AccessorialMeasures{
		Miles:  sp.RatingMiles(),
		Dwells: sp.DwellHours(),
		Days:   sp.ServiceDays(),
		Stops:  decimal.NewFromInt(int64(sp.RatedStopCount())),
	}
}

func PriceAgreementAccessorials(
	ctx context.Context,
	measures *rateagreement.AccessorialMeasures,
	accessorials []*rateagreement.RateAgreementAccessorial,
	freight decimal.Decimal,
) ([]PricedAgreementAccessorial, error) {
	out := make([]PricedAgreementAccessorial, 0, len(accessorials))
	percentages := make([]int, 0, len(accessorials))

	for _, accessorial := range accessorials {
		if accessorial == nil {
			continue
		}
		charge := &AdditionalCharge{Unit: 1, AccessorialChargeID: accessorial.AccessorialChargeID}
		switch accessorial.Method {
		case accessorialcharge.MethodPercentage:
			charge.Method = accessorialcharge.MethodPercentage
			charge.Amount = accessorial.PricedAmount()
			charge.Rate = decimal.NewNullDecimal(accessorial.Amount)
			percentages = append(percentages, len(out))
		case accessorialcharge.MethodPerUnit:
			quantity := accessorial.BillableQuantity(measures)
			amount, _ := accessorial.Cap(
				accessorial.PricedAmount().Mul(quantity).Round(centsPlaces),
			)
			charge.Method = accessorialcharge.MethodFlat
			charge.Amount = amount
			charge.Quantity = decimal.NewNullDecimal(quantity)
			charge.Rate = decimal.NewNullDecimal(accessorial.Amount)
			charge.RateUnit = accessorial.TariffRateUnit()
		default:
			amount, _ := accessorial.Cap(accessorial.PricedAmount())
			charge.Method = accessorialcharge.MethodFlat
			charge.Amount = amount
			charge.Quantity = decimal.NewNullDecimal(decimal.NewFromInt(1))
		}
		out = append(out, PricedAgreementAccessorial{Accessorial: accessorial, Charge: charge})
	}

	if err := capPercentages(ctx, out, percentages, freight); err != nil {
		return nil, err
	}

	kept := out[:0]
	for _, priced := range out {
		if priced.Charge.Amount.IsPositive() {
			kept = append(kept, priced)
		}
	}
	return kept, nil
}

func capPercentages(
	ctx context.Context,
	priced []PricedAgreementAccessorial,
	percentages []int,
	freight decimal.Decimal,
) error {
	if len(percentages) == 0 {
		return nil
	}
	charges := make([]*AdditionalCharge, 0, len(percentages))
	for _, index := range percentages {
		charges = append(charges, priced[index].Charge)
	}
	totals, err := AdditionalChargeTotals(ctx, charges, freight)
	if err != nil {
		return err
	}
	for position, index := range percentages {
		entry := priced[index]
		capped, applied := entry.Accessorial.Cap(totals.Each[position])
		if !applied {
			continue
		}
		entry.Charge.Method = accessorialcharge.MethodFlat
		entry.Charge.Amount = capped
	}
	return nil
}
