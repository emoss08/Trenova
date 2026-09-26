package agenttoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

var (
	uncancelFields = []string{fieldStatus, fieldCanceledAt, fieldCanceledByID, paramCancelReason}
	rerateFields   = []string{
		fieldFormulaTemplateID, "baseRate", fieldFreightChargeAmount, "otherChargeAmount",
		"totalChargeAmount",
	}
)

func renderUncancel(
	_ *repositories.UncancelShipmentRequest,
	plan *serviceports.ShipmentChangePreview,
) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(
		shipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(uncancelFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldCanceledByID: permission.ResourceUser,
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
	_ *repositories.TransferOwnershipRequest,
	plan *serviceports.ShipmentChangePreview,
) (*agent.ToolPreview, error) {
	change, err := toolpreview.Changed(
		shipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(paramOwnerID),
		toolpreview.WithRefs(map[string]permission.Resource{
			paramOwnerID: permission.ResourceUser,
		}),
	)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(fmt.Sprintf(
		"Would hand shipment %s to a new owner.", plan.Before.ProNumber,
	), change), nil
}

func renderRerate(
	_ *serviceports.AutoRateShipmentRequest,
	plan *serviceports.ShipmentAutoRatePreview,
) (*agent.ToolPreview, error) {
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
		shipmentRecord(plan.Before),
		plan.Before,
		plan.After,
		toolpreview.Only(rerateFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldFormulaTemplateID: permission.ResourceFormulaTemplate,
		}),
	)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		change,
		chargeEditMoney(plan.Before, plan.After),
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
	plan *serviceports.ShipmentDistancePreview,
) (*agent.ToolPreview, error) {
	before := make(map[pulid.ID]*shipment.ShipmentMove, len(plan.Before.Moves))
	for _, move := range plan.Before.Moves {
		if move != nil {
			before[move.ID] = move
		}
	}

	changes := make([]*agent.RecordChange, 0, len(plan.After.Moves))
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
				Version:  moveVersion(move),
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
		countOf(len(changes), "move"),
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

type duplicateCopyView struct {
	Source            string              `json:"source"`
	CustomerID        pulid.ID            `json:"customerId"`
	BOL               string              `json:"bol"`
	FreightTerms      string              `json:"freightTerms"`
	FormulaTemplateID pulid.ID            `json:"formulaTemplateId"`
	FirstPickupAt     int64               `json:"firstPickupAt"`
	LastDeliveryAt    int64               `json:"lastDeliveryAt"`
	Stops             int                 `json:"stops"`
	Commodities       int                 `json:"commodities"`
	Charges           int                 `json:"charges"`
	TotalChargeAmount decimal.NullDecimal `json:"totalChargeAmount"`
}

func duplicateViewOf(source string, copied *shipment.Shipment) *duplicateCopyView {
	view := &duplicateCopyView{
		Source:            source,
		CustomerID:        copied.CustomerID,
		BOL:               copied.BOL,
		FreightTerms:      string(copied.FreightTerms),
		FormulaTemplateID: copied.FormulaTemplateID,
		Commodities:       len(copied.Commodities),
		Charges:           len(copied.AdditionalCharges),
		TotalChargeAmount: copied.TotalChargeAmount,
	}
	for _, move := range copied.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			view.Stops++
			if view.FirstPickupAt == 0 {
				view.FirstPickupAt = stop.ScheduledWindowStart
			}
			view.LastDeliveryAt = stop.ScheduledWindowStart
		}
	}

	return view
}

func renderDuplicate(
	req *repositories.BulkDuplicateShipmentRequest,
	plan *serviceports.ShipmentDuplicatePreview,
) (*agent.ToolPreview, error) {
	changes := make([]*agent.RecordChange, 0, len(plan.Copies))
	for idx, copied := range plan.Copies {
		if copied == nil {
			continue
		}
		change, err := shipmentCopyChange(
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
		countOf(req.Count, "new shipment"),
		dates,
	), changes...), nil
}

func shipmentCopyChange(
	label, source string,
	copied *shipment.Shipment,
) (*agent.RecordChange, error) {
	return toolpreview.Create(
		toolpreview.Record{Resource: permission.ResourceShipment, Label: label},
		duplicateViewOf(source, copied),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldCustomerID:        permission.ResourceCustomer,
			fieldFormulaTemplateID: permission.ResourceFormulaTemplate,
		}),
		toolpreview.Types(operationDateTimes),
		toolpreview.SensitiveAs("totalChargeAmount"),
	)
}
