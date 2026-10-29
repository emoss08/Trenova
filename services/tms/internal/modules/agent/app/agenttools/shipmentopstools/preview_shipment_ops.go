package shipmentopstools

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var uncancelFields = []string{
	changepreview.FieldStatus,
	changepreview.FieldCanceledAt,
	changepreview.FieldCanceledByID,
	changepreview.FieldCancelReason,
}

var rerateFields = []string{
	changepreview.FieldFormulaTemplateID, "tariffInputs", changepreview.FieldFreightChargeAmount,
	"otherChargeAmount",
	"totalChargeAmount",
}

func renderUncancel(
	_ *shipmentports.UncancelShipmentRequest,
	plan *shipmentserviceports.ShipmentChangePreview,
) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(
		shipmentpreview.ShipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(uncancelFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldCanceledByID: permission.ResourceUser,
		}),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would reopen canceled shipment %s as New, with its moves and stops back to New. "+
			"Nothing is sent and nobody is assigned.",
		plan.Before.ProNumber,
	), change), nil
}

func renderOwnershipTransfer(
	_ *shipmentports.TransferOwnershipRequest,
	plan *shipmentserviceports.ShipmentChangePreview,
) (*agentpreview.ToolPreview, error) {
	change, err := toolpreview.Changed(
		shipmentpreview.ShipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(toolparams.ParamOwnerID),
		toolpreview.WithRefs(map[string]permission.Resource{
			toolparams.ParamOwnerID: permission.ResourceUser,
		}),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would hand shipment %s to a new owner.", plan.Before.ProNumber,
	), change), nil
}

type reratePlan struct {
	*shipmentserviceports.ShipmentAutoRatePreview
	money *agentpreview.MoneyPreview
}

func renderRerate(
	_ *shipmentserviceports.AutoRateShipmentRequest,
	plan *reratePlan,
) (*agentpreview.ToolPreview, error) {
	application := plan.Application
	if application == nil || !application.Applied {
		outcome := "no agreement covers it"
		if application != nil {
			outcome = "no agreement covers it (" + string(application.Outcome) + ")"
		}

		return toolpreview.Build(fmt.Sprintf(
			"Would leave shipment %s as it is: %s.", plan.Before.ProNumber, outcome,
		)), nil
	}

	change, err := toolpreview.Changed(
		shipmentpreview.ShipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(rerateFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldFormulaTemplateID: permission.ResourceFormulaTemplate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		plan.money,
		toolpreview.SensitiveAs("totalChargeAmount", "otherChargeAmount"),
	)

	source := strings.TrimSpace(application.AgreementName)
	if rule := strings.TrimSpace(application.RuleLabel); rule != "" {
		source = strings.TrimSpace(source + " (" + rule + ")")
	}
	if source == "" {
		source = "its rate agreement"
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would price shipment %s again from %s, overwriting its rating method, base rate "+
			"and contract accessorials.",
		plan.Before.ProNumber,
		source,
	), change), nil
}

type moveDistanceView struct {
	Distance *float64 `json:"distance"`
}

func renderDistance(
	_ *distanceRequest,
	plan *shipmentdistanceports.ShipmentDistancePreview,
) (*agentpreview.ToolPreview, error) {
	before := make(map[pulid.ID]*shipment.ShipmentMove, len(plan.Before.Moves))
	for _, move := range plan.Before.Moves {
		if move != nil {
			before[move.ID] = move
		}
	}

	changes := make([]*agentpreview.RecordChange, 0, len(plan.After.Moves))
	for _, move := range plan.After.Moves {
		if move == nil {
			continue
		}
		previous, ok := before[move.ID]
		if !ok || sameDistance(previous.Distance, move.Distance) {
			continue
		}
		change, err := toolpreview.Changed(
			toolpreview.Record{
				Resource: permission.ResourceShipmentMove,
				ID:       move.ID,
				Label:    fmt.Sprintf("%s move %d", plan.Before.ProNumber, move.Sequence+1),
				Version:  shipmentpreview.MoveVersion(move),
			},
			&moveDistanceView{Distance: previous.Distance},
			&moveDistanceView{Distance: move.Distance},
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	total := 0.0
	if plan.Distance != nil {
		total = plan.Distance.TotalDistance
	}
	if len(changes) == 0 {
		return toolpreview.Build(fmt.Sprintf(
			"Would work out shipment %s's miles again and find them unchanged at %.1f.",
			plan.Before.ProNumber,
			total,
		)), nil
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would save new miles on %s of shipment %s, %.1f in all.",
		changepreview.CountOf(len(changes), "move"),
		plan.Before.ProNumber,
		total,
	), changes...), nil
}

func sameDistance(before, after *float64) bool {
	switch {
	case before == nil && after == nil:
		return true
	case before == nil || after == nil:
		return false
	default:
		return decimal.NewFromFloat(*before).Round(2).Equal(decimal.NewFromFloat(*after).Round(2))
	}
}

func renderDuplicate(
	req *shipmentports.BulkDuplicateShipmentRequest,
	plan *shipmentserviceports.ShipmentDuplicatePreview,
) (*agentpreview.ToolPreview, error) {
	changes := make([]*agentpreview.RecordChange, 0, len(plan.Copies))
	for idx, copied := range plan.Copies {
		if copied == nil {
			continue
		}
		change, err := shipmentpreview.ShipmentCopyChange(
			fmt.Sprintf("Copy %d of %s", idx+1, plan.Source.ProNumber),
			plan.Source.ProNumber,
			copied,
		)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	dates := "keeping its dates"
	if req.FirstPickupAt != nil {
		dates = "moving every stop so the first pickup starts at the time given"
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would copy shipment %s into %s, %s. Each copy gets its own pro number and order.",
		plan.Source.ProNumber,
		changepreview.CountOf(req.Count, "new shipment"),
		dates,
	), changes...), nil
}
