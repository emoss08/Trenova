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
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var _ serviceports.ToolPreviewer = (*createShipmentTool)(nil)

var enteredShipmentFields = []string{
	fieldCustomerID,
	fieldBillToCustomerID,
	fieldServiceTypeID,
	fieldShipmentTypeID,
	fieldFormulaTemplateID,
	fieldStatus,
	fieldBol,
	"freightTerms",
	fieldPieces,
	fieldWeight,
	fieldTemperatureMin,
	fieldTemperatureMax,
}

var enteredShipmentRefs = map[string]permission.Resource{
	fieldCustomerID:        permission.ResourceCustomer,
	fieldBillToCustomerID:  permission.ResourceCustomer,
	fieldServiceTypeID:     permission.ResourceServiceType,
	fieldShipmentTypeID:    permission.ResourceShipmentType,
	fieldFormulaTemplateID: permission.ResourceFormulaTemplate,
}

var enteredShipmentLabels = map[string]string{
	fieldCustomerID:        labelCustomer,
	fieldBillToCustomerID:  "Bill to",
	fieldServiceTypeID:     "Service type",
	fieldShipmentTypeID:    "Shipment type",
	fieldFormulaTemplateID: "Rating method",
	fieldBol:               "BOL",
}

var enteredStopFields = []string{
	fieldType,
	fieldLocationID,
	"scheduleType",
	fieldScheduledWindowStart,
	fieldScheduledWindowEnd,
	fieldPieces,
	fieldWeight,
}

var enteredStopLabels = map[string]string{
	fieldLocationID:           "Location",
	fieldScheduledWindowStart: "Window opens",
	fieldScheduledWindowEnd:   "Window closes",
}

var shipmentChargeFields = []string{"freightChargeAmount", "otherChargeAmount"}

func enteredShipmentOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(enteredShipmentFields...),
		toolpreview.WithRefs(enteredShipmentRefs),
		toolpreview.Labels(enteredShipmentLabels),
	}
}

func enteredStopOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(enteredStopFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			fieldLocationID: permission.ResourceLocation,
		}),
		toolpreview.Labels(enteredStopLabels),
	}
}

func (t *createShipmentTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := guardPreview(t, &params); err != nil {
		return nil, err
	}

	entity, err := t.draft(&params)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build("Would enter a new shipment."), err), nil
		}

		return nil, err
	}

	plan, err := t.shipments.PreviewCreate(ctx, entity, params.Actor)
	if err != nil {
		if isRefusal(err) {
			return warnWouldFail(toolpreview.Build(enteredShipmentSummary(entity, nil)), err), nil
		}

		return nil, err
	}

	changes, err := enteredShipmentChanges(plan)
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(enteredShipmentSummary(plan.Shipment, plan.Rating), changes...), nil
}

func enteredShipmentChanges(plan *serviceports.ShipmentCreatePlan) ([]*agent.RecordChange, error) {
	entity := plan.Shipment
	label := "New shipment"
	if entity.BOL != "" {
		label = "Shipment with BOL " + entity.BOL
	}

	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipment,
		Label:    label,
	}, entity, enteredShipmentOptions()...)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		created,
		enteredShipmentMoney(entity, plan.Rating),
		toolpreview.SensitiveAs(shipmentChargeFields...),
	)

	changes := []*agent.RecordChange{created}
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			stopChange, sErr := toolpreview.Create(toolpreview.Record{
				Resource: permission.ResourceShipmentStop,
				Label:    fmt.Sprintf("%s stop %d", stop.Type, stop.Sequence+1),
			}, stop, enteredStopOptions()...)
			if sErr != nil {
				return nil, sErr
			}
			changes = append(changes, stopChange)
		}
	}

	return changes, nil
}

func enteredShipmentMoney(
	entity *shipment.Shipment,
	rating *serviceports.ShipmentRatingOutcome,
) *agent.MoneyPreview {
	currency := money.DefaultCurrencyCode
	if rating != nil && rating.Currency != "" {
		currency = rating.Currency
	}

	return toolpreview.MoneyBlock(
		currency,
		agent.MoneyLine{Label: "Linehaul", After: entity.FreightChargeAmount},
		agent.MoneyLine{Label: "Other charges", After: entity.OtherChargeAmount},
	)
}

func enteredShipmentSummary(
	entity *shipment.Shipment,
	rating *serviceports.ShipmentRatingOutcome,
) string {
	stops := 0
	for _, move := range entity.Moves {
		if move != nil {
			stops += len(move.Stops)
		}
	}

	summary := fmt.Sprintf(
		"Would enter a new shipment with %d %s",
		stops,
		stringutils.Pluralize("stop", "stops", stops),
	)
	if entity.BOL != "" {
		summary += " under BOL " + entity.BOL
	}
	summary += "."

	if total := entity.TotalChargeAmount; total.Valid && !total.Decimal.Equal(decimal.Zero) {
		summary += fmt.Sprintf(" It would charge %s in total", total.Decimal.StringFixed(2))
		if rating != nil && rating.AgreementName != "" {
			summary += " under " + rating.AgreementName
		}
		summary += "."
	}
	if rating == nil {
		return summary
	}
	if !rating.Adopted {
		summary += fmt.Sprintf(
			" The rate agreement would charge %s %s for the linehaul; the shipment keeps the rate entered.",
			rating.Amount.StringFixed(2),
			rating.Currency,
		)
	}
	if explanation := strings.TrimRight(
		strings.TrimSpace(rating.Explanation),
		".",
	); explanation != "" {
		summary += " " + explanation + "."
	}

	return summary
}
