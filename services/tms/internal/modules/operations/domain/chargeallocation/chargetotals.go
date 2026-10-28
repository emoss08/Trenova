package chargeallocation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

type ChargeTotals struct {
	Each []decimal.Decimal
	Sum  decimal.Decimal
}

func AdditionalChargeTotals(
	ctx context.Context,
	charges []*shipment.AdditionalCharge,
	baseCharge decimal.Decimal,
) (*ChargeTotals, error) {
	if len(charges) == 0 {
		return &ChargeTotals{Sum: decimal.Zero}, nil
	}

	req := &rules.AllocationChargesRequest{
		Base:    baseCharge,
		Charges: make([]rules.AllocationCharge, 0, len(charges)),
	}
	for _, charge := range charges {
		req.Charges = append(req.Charges, accessorialCharge(charge, nil))
	}

	totals, err := rules.ChargeTotals(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("price additional charges: %w", err)
	}
	if len(totals.Totals) != len(charges) {
		return nil, fmt.Errorf(
			"%w: %d totals came back for %d charges",
			errSplitShape,
			len(totals.Totals),
			len(charges),
		)
	}

	return &ChargeTotals{Each: totals.Totals, Sum: totals.Sum}, nil
}

func ChargeTotalsByCharge(
	ctx context.Context,
	shp *shipment.Shipment,
) (map[*shipment.AdditionalCharge]decimal.Decimal, error) {
	if shp == nil {
		return map[*shipment.AdditionalCharge]decimal.Decimal{}, nil
	}

	totals, err := AdditionalChargeTotals(
		ctx,
		shp.AdditionalCharges,
		shp.FreightChargeAmount.Decimal,
	)
	if err != nil {
		return nil, err
	}

	byCharge := make(map[*shipment.AdditionalCharge]decimal.Decimal, len(shp.AdditionalCharges))
	for i, charge := range shp.AdditionalCharges {
		if charge != nil {
			byCharge[charge] = totals.Each[i]
		}
	}

	return byCharge, nil
}
