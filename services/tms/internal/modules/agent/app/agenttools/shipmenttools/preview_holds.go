package shipmenttools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentholdports"
)

var placedHoldFields = []string{
	changepreview.FieldShipmentID,
	changepreview.FieldType,
	changepreview.FieldSeverity,
	"reasonCode",
	changepreview.FieldNotes,
	"blocksDispatch",
	"blocksDelivery",
	"blocksBilling",
	"visibleToCustomer",
	"startedAt",
}

var placedHoldLabels = map[string]string{
	changepreview.FieldShipmentID: changepreview.LabelShipment,
	"reasonCode":                  "Reason",
}

var ReleasedHoldFields = []string{"releasedAt"}

func PlacedHoldOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(placedHoldFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldShipmentID: permission.ResourceShipment,
		}),
		toolpreview.Labels(placedHoldLabels),
		toolpreview.Volatile("startedAt"),
	}
}

func holdLabel(hold *shipmenthold.ShipmentHold) string {
	return strings.TrimSpace(fmt.Sprintf("%s hold %s", hold.Type, hold.ReasonCode))
}

func holdBlocks(hold *shipmenthold.ShipmentHold) string {
	blocks := make([]string, 0, 3)
	if hold.BlocksDispatch {
		blocks = append(blocks, "dispatch")
	}
	if hold.BlocksDelivery {
		blocks = append(blocks, "delivery")
	}
	if hold.BlocksBilling {
		blocks = append(blocks, "billing")
	}
	if len(blocks) == 0 {
		return "blocks nothing and is advisory"
	}

	return "blocks " + strings.Join(blocks, ", ")
}

func (t *PlaceShipmentHoldTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	hold, err := t.holds.PreviewCreate(ctx, request, params.Actor)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would put a shipment on hold."),
				err,
			), nil
		}

		return nil, err
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		Label:    holdLabel(hold),
	}, hold, PlacedHoldOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would put the shipment on a %s hold that %s until someone clears it.",
		strings.ToLower(string(hold.Severity)),
		holdBlocks(hold),
	), change), nil
}

func (t *ReleaseShipmentHoldTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	request, err := t.request(&params)
	if err != nil {
		return nil, err
	}

	current, err := t.holds.GetByID(ctx, &shipmentholdports.GetShipmentHoldByIDRequest{
		HoldID:     request.HoldID,
		ShipmentID: request.ShipmentID,
		TenantInfo: request.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf(
		"Would release the %s, which %s, so the shipment can move again.",
		holdLabel(current),
		holdBlocks(current),
	)
	released, err := t.holds.PreviewRelease(ctx, request, params.Actor)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		ID:       current.ID,
		Label:    holdLabel(current),
		Version:  changepreview.PreviewVersion(current.Version),
	}, current, released, toolpreview.Only(ReleasedHoldFields...), toolpreview.Volatile("releasedAt"))
	if err != nil {
		return nil, err
	}
	change.Operation = agentpreview.PreviewOperationArchive

	return toolpreview.Build(summary, change), nil
}
