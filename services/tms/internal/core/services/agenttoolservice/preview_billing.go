package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ serviceports.ToolPreviewer = (*correctChargeCodeTool)(nil)

func (t *correctChargeCodeTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	plan, err := t.billing.PreviewUpdateCharges(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceShipment,
			ID:       plan.Before.ID,
			Label:    plan.Before.ProNumber,
			Version:  pinnedVersion(plan.Before.Version),
		},
		plan.Before,
		plan.After,
		toolpreview.Only("otherChargeAmount", "totalChargeAmount"),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		chargeEditMoney(plan.Before, plan.After),
		toolpreview.SensitiveAs("totalChargeAmount", "otherChargeAmount"),
	)

	summary := fmt.Sprintf(
		"Would replace the additional charges of billing queue item %s on shipment %s "+
			"with %s, taking its total from %s to %s.",
		plan.Item.Number,
		plan.Before.ProNumber,
		countOf(len(plan.After.AdditionalCharges), "charge"),
		plan.Before.TotalChargeAmount.Decimal.StringFixed(2),
		plan.After.TotalChargeAmount.Decimal.StringFixed(2),
	)
	preview := toolpreview.Build(summary, change)
	if plan.ConvertedSplits > 0 {
		toolpreview.Warn(preview, agent.PreviewWarningWouldFail, fmt.Sprintf(
			"%d split(s) by amount would be rewritten as percentages.", plan.ConvertedSplits,
		))
	}

	return preview, nil
}

// chargeEditMoney is the shipment's charges line by line: the freight, then
// every additional charge as it was and as the edit leaves it, a removed one
// with nothing after and an added one with nothing before. The totals are the
// shipment's total charge on either side.
func chargeEditMoney(before, after *shipment.Shipment) *agent.MoneyPreview {
	freightBefore := before.FreightChargeAmount.Decimal
	freightAfter := after.FreightChargeAmount.Decimal
	names := accessorialLabels(before.AdditionalCharges, after.AdditionalCharges)

	lines := make(
		[]agent.MoneyLine,
		0,
		len(before.AdditionalCharges)+len(after.AdditionalCharges)+1,
	)
	lines = append(lines, agent.MoneyLine{
		Label:  "Freight",
		Before: knownAmount(freightBefore),
		After:  knownAmount(freightAfter),
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
			Before: knownAmount(charge.Total(freightBefore)),
		}
		if edited, ok := kept[charge.ID]; ok {
			line.Label = names[edited.AccessorialChargeID]
			line.After = knownAmount(edited.Total(freightAfter))
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
			After: knownAmount(charge.Total(freightAfter)),
		})
	}

	block := toolpreview.MoneyBlock(money.DefaultCurrencyCode, lines...)
	block.TotalBefore = before.TotalChargeAmount
	block.TotalAfter = after.TotalChargeAmount
	block.Delta = toolpreview.MoneyDelta(block.TotalBefore, block.TotalAfter)

	return block
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
