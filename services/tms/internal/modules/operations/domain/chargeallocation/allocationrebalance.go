package chargeallocation

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type StaleAmountSplit struct {
	Kind               shipmentvocab.ChargeAllocationKind
	AdditionalChargeID pulid.ID
	ChargeIndex        int
	AllocatedTotal     decimal.Decimal
	ChargeTotal        decimal.Decimal
}

type splitConversion struct {
	rows     []*shipment.ChargeAllocation
	percents []decimal.NullDecimal
}

type AmountSplitRebalance struct {
	Stale       []StaleAmountSplit
	conversions []splitConversion
	converted   int
}

func (r *AmountSplitRebalance) HasStale() bool {
	return r != nil && len(r.Stale) > 0
}

func (r *AmountSplitRebalance) ConvertToPercent() int {
	if r == nil {
		return 0
	}
	for _, conversion := range r.conversions {
		for i, row := range conversion.rows {
			if i >= len(conversion.percents) || !conversion.percents[i].Valid {
				continue
			}
			row.Method = shipmentvocab.ChargeAllocationMethodPercent
			row.Percent = conversion.percents[i]
			row.Amount = decimal.NullDecimal{}
		}
	}

	return r.converted
}

type splitGroup struct {
	kind        shipmentvocab.ChargeAllocationKind
	chargeID    pulid.ID
	chargeIndex int
	rows        []*shipment.ChargeAllocation
}

func RebalanceAmountSplits(
	ctx context.Context,
	shp *shipment.Shipment,
	allocations []*shipment.ChargeAllocation,
) (*AmountSplitRebalance, error) {
	if shp == nil {
		return &AmountSplitRebalance{}, nil
	}

	freightRows, byCharge := groupAllocations(allocations)
	groups := make([]splitGroup, 0, len(byCharge)+1)
	req := &rules.AllocationChargesRequest{
		Base:    shp.FreightChargeAmount.Decimal,
		Charges: make([]rules.AllocationCharge, 0, len(byCharge)+1),
	}
	if len(freightRows) > 0 {
		groups = append(groups, splitGroup{
			kind:        shipmentvocab.ChargeAllocationKindFreight,
			chargeIndex: -1,
			rows:        freightRows,
		})
		req.Charges = append(req.Charges, totalCharge(
			rules.AllocationKindFreight,
			shp.FreightChargeAmount.Decimal,
			freightRows,
		))
	}
	for i, charge := range shp.AdditionalCharges {
		if charge == nil || len(byCharge[charge.ID]) == 0 {
			continue
		}
		rows := byCharge[charge.ID]
		groups = append(groups, splitGroup{
			kind:        shipmentvocab.ChargeAllocationKindAccessorial,
			chargeID:    charge.ID,
			chargeIndex: i,
			rows:        rows,
		})
		req.Charges = append(req.Charges, accessorialCharge(charge, rows))
	}
	if len(groups) == 0 {
		return &AmountSplitRebalance{}, nil
	}

	out, err := rules.RebalanceCharges(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("check amount splits: %w", err)
	}
	if len(out.Charges) != len(groups) {
		return nil, fmt.Errorf(
			"%w: %d charges came back for %d sent",
			errSplitShape,
			len(out.Charges),
			len(groups),
		)
	}

	rebalance := &AmountSplitRebalance{
		Stale:       make([]StaleAmountSplit, 0, len(groups)),
		conversions: make([]splitConversion, 0, len(groups)),
		converted:   out.Converted,
	}
	for i, charge := range out.Charges {
		if !charge.Stale {
			continue
		}
		group := groups[i]
		rebalance.Stale = append(rebalance.Stale, StaleAmountSplit{
			Kind:               group.kind,
			AdditionalChargeID: group.chargeID,
			ChargeIndex:        group.chargeIndex,
			AllocatedTotal:     charge.AllocatedTotal.Decimal,
			ChargeTotal:        charge.ChargeTotal.Decimal,
		})
		if charge.Percents != nil {
			rebalance.conversions = append(rebalance.conversions, splitConversion{
				rows:     group.rows,
				percents: charge.Percents,
			})
		}
	}

	return rebalance, nil
}
