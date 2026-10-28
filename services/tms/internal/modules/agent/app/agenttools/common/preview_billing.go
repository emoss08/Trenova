package common

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/chargeallocation"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

// chargeEditMoney is the shipment's charges line by line: the freight, then
// every additional charge as it was and as the edit leaves it, a removed one
// with nothing after and an added one with nothing before. The totals are the
// shipment's total charge on either side.
func ChargeEditMoney(
	ctx context.Context,
	before, after *shipment.Shipment,
) (*agent.MoneyPreview, error) {
	freightBefore := before.FreightChargeAmount.Decimal
	freightAfter := after.FreightChargeAmount.Decimal
	totalsBefore, err := chargeallocation.ChargeTotalsByCharge(ctx, before)
	if err != nil {
		return nil, err
	}
	totalsAfter, err := chargeallocation.ChargeTotalsByCharge(ctx, after)
	if err != nil {
		return nil, err
	}
	names := accessorialLabels(before.AdditionalCharges, after.AdditionalCharges)

	lines := make(
		[]agent.MoneyLine,
		0,
		len(before.AdditionalCharges)+len(after.AdditionalCharges)+1,
	)
	lines = append(lines, agent.MoneyLine{
		Label:  "Freight",
		Before: KnownAmount(freightBefore),
		After:  KnownAmount(freightAfter),
	})

	kept := make(map[pulid.ID]*shipment.AdditionalCharge, len(after.AdditionalCharges))
	for _, charge := range after.AdditionalCharges {
		if charge != nil && charge.ID.IsNotNil() {
			kept[charge.ID] = charge
		}
	}

	for _, charge := range before.AdditionalCharges {
		if charge == nil {
			continue
		}
		line := agent.MoneyLine{
			Label:  names[charge.AccessorialChargeID],
			Before: KnownAmount(totalsBefore[charge]),
		}
		if edited, ok := kept[charge.ID]; ok {
			line.Label = names[edited.AccessorialChargeID]
			line.After = KnownAmount(totalsAfter[edited])
		}
		lines = append(lines, line)
	}

	existing := make(map[pulid.ID]struct{}, len(before.AdditionalCharges))
	for _, charge := range before.AdditionalCharges {
		if charge != nil {
			existing[charge.ID] = struct{}{}
		}
	}
	for _, charge := range after.AdditionalCharges {
		if charge == nil {
			continue
		}
		if _, ok := existing[charge.ID]; ok && charge.ID.IsNotNil() {
			continue
		}
		lines = append(lines, agent.MoneyLine{
			Label: names[charge.AccessorialChargeID],
			After: KnownAmount(totalsAfter[charge]),
		})
	}

	block := toolpreview.MoneyBlock(money.DefaultCurrencyCode, lines...)
	block.TotalBefore = before.TotalChargeAmount
	block.TotalAfter = after.TotalChargeAmount
	block.Delta = toolpreview.MoneyDelta(block.TotalBefore, block.TotalAfter)

	return block, nil
}

// accessorialLabels names each accessorial by the description the shipment's
// charges carry. The edited set comes from the call and carries none, so an
// accessorial only the edit introduces reads by its kind.
func accessorialLabels(sets ...[]*shipment.AdditionalCharge) map[pulid.ID]string {
	names := make(map[pulid.ID]string)
	for _, set := range sets {
		for _, charge := range set {
			if charge == nil {
				continue
			}
			if _, named := names[charge.AccessorialChargeID]; named {
				continue
			}
			names[charge.AccessorialChargeID] = accessorialLabel(charge)
		}
	}

	return names
}

func accessorialLabel(charge *shipment.AdditionalCharge) string {
	if charge.AccessorialCharge != nil {
		if description := strings.TrimSpace(
			charge.AccessorialCharge.Description,
		); description != "" {
			return description
		}
		if code := strings.TrimSpace(charge.AccessorialCharge.Code); code != "" {
			return code
		}
	}
	if charge.IsDetention {
		return "Detention"
	}

	return "Accessorial charge"
}
