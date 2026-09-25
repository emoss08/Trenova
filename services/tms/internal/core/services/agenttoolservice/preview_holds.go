package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var (
	_ serviceports.ToolPreviewer = (*placeShipmentHoldTool)(nil)
	_ serviceports.ToolPreviewer = (*releaseShipmentHoldTool)(nil)
)

var placedHoldFields = []string{
	"shipmentId",
	"type",
	"severity",
	"reasonCode",
	"notes",
	"blocksDispatch",
	"blocksDelivery",
	"blocksBilling",
	"visibleToCustomer",
	"startedAt",
}

var placedHoldLabels = map[string]string{"shipmentId": "Shipment", "reasonCode": "Reason"}

var releasedHoldFields = []string{"releasedAt"}

func placedHoldOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(placedHoldFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			"shipmentId": permission.ResourceShipment,
		}),
		toolpreview.Labels(placedHoldLabels),
		toolpreview.Volatile("startedAt"),
	}
}

func holdLabel(hold *shipment.ShipmentHold) string {
	return strings.TrimSpace(fmt.Sprintf("%s hold %s", hold.Type, hold.ReasonCode))
}

func holdBlocks(hold *shipment.ShipmentHold) string {
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

func (t *placeShipmentHoldTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	hold, err := t.holds.PreviewCreate(ctx, request, params.Actor)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would put a shipment on hold."), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		Label:    holdLabel(hold),
	}, hold, placedHoldOptions()...)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would put the shipment on a %s hold that %s until someone clears it.",
		strings.ToLower(string(hold.Severity)),
		holdBlocks(hold),
	), change), nil
}

func (t *releaseShipmentHoldTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	request, err := t.request(params)
	if err != nil {
		return nil, err
	}

	current, err := t.holds.GetByID(ctx, &repositories.GetShipmentHoldByIDRequest{
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
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(summary), err), nil
		}

		return nil, err
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceShipmentHold,
		ID:       current.ID,
		Label:    holdLabel(current),
		Version:  previewVersion(current.Version),
	}, current, released, toolpreview.Only(releasedHoldFields...), toolpreview.Volatile("releasedAt"))
	if err != nil {
		return nil, err
	}
	change.Operation = agent.PreviewOperationArchive

	return toolpreview.Build(summary, change), nil
}
