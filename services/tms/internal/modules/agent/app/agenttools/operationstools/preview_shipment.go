package operationstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

func (t *UpdateShipmentTool) Preview(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	before, after, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		common.ShipmentRecord(before),
		before,
		after,
		toolpreview.Only(
			common.PreviewFieldCustomerID,
			common.PreviewFieldServiceTypeID,
			common.PreviewFieldShipmentTypeID,
			common.PreviewFieldTractorTypeID,
			common.PreviewFieldTrailerTypeID,
			"bol",
			"pieces",
			"weight",
			"temperatureMin",
			"temperatureMax",
		),
		toolpreview.WithRefs(common.UpdateShipmentRefs),
	)
	if err != nil {
		return nil, err
	}

	notices, err := common.TenderChangeNotices(ctx, t.partners, before, after, params.Actor)
	if err != nil {
		return nil, err
	}
	changes := append([]*agent.RecordChange{change}, notices...)

	summary := fmt.Sprintf("Would change %s on shipment %s.",
		common.CountOf(len(change.Fields), "field"), before.ProNumber)
	if partners := len(changes) - 1; partners > 0 {
		summary += fmt.Sprintf(" %s tendered the load would be sent an EDI 204 change.",
			common.CountOf(partners, "partner"))
	}

	return toolpreview.Build(summary, changes...), nil
}
