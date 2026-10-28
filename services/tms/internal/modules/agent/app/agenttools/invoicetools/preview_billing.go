package invoicetools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/invoicepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

func (t *CorrectChargeCodeTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
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
			Version:  changepreview.PinnedVersion(plan.Before.Version),
		},
		plan.Before,
		plan.After,
		toolpreview.Only("otherChargeAmount", "totalChargeAmount"),
	)
	if err != nil {
		return nil, err
	}
	money, err := invoicepreview.ChargeEditMoney(ctx, plan.Before, plan.After)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		money,
		toolpreview.SensitiveAs("totalChargeAmount", "otherChargeAmount"),
	)

	summary := fmt.Sprintf(
		"Would replace the additional charges of billing queue item %s on shipment %s "+
			"with %s, taking its total from %s to %s.",
		plan.Item.Number,
		plan.Before.ProNumber,
		changepreview.CountOf(len(plan.After.AdditionalCharges), "charge"),
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
