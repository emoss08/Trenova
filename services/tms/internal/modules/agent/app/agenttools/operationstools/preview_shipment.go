package operationstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

func (t *UpdateShipmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	before, after, err := t.plan(ctx, &params)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		shipmentpreview.ShipmentRecord(before),
		before,
		after,
		toolpreview.Only(
			changepreview.PreviewFieldCustomerID,
			changepreview.PreviewFieldServiceTypeID,
			changepreview.PreviewFieldShipmentTypeID,
			changepreview.PreviewFieldTractorTypeID,
			changepreview.PreviewFieldTrailerTypeID,
			"bol",
			"pieces",
			"weight",
			"temperatureMin",
			"temperatureMax",
		),
		toolpreview.WithRefs(shipmentpreview.UpdateShipmentRefs),
	)
	if err != nil {
		return nil, err
	}

	notices, err := shipmentpreview.TenderChangeNotices(
		ctx,
		t.partners,
		before,
		after,
		params.Actor,
	)
	if err != nil {
		return nil, err
	}
	changes := append([]*agentpreview.RecordChange{change}, notices...)

	summary := fmt.Sprintf("Would change %s on shipment %s.",
		changepreview.CountOf(len(change.Fields), "field"), before.ProNumber)
	if partners := len(changes) - 1; partners > 0 {
		summary += fmt.Sprintf(" %s tendered the load would be sent an EDI 204 change.",
			changepreview.CountOf(partners, "partner"))
	}

	return toolpreview.Build(summary, changes...), nil
}
