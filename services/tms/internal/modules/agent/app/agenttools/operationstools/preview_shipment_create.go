package operationstools

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
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/money"
	"github.com/shopspring/decimal"
)

var enteredShipmentFields = []string{
	changepreview.FieldCustomerID,
	changepreview.FieldBillToCustomerID,
	changepreview.FieldServiceTypeID,
	changepreview.FieldShipmentTypeID,
	changepreview.FieldFormulaTemplateID,
	changepreview.FieldStatus,
	changepreview.FieldBol,
	"freightTerms",
	changepreview.FieldPieces,
	changepreview.FieldWeight,
	changepreview.FieldTemperatureMin,
	changepreview.FieldTemperatureMax,
}

var enteredShipmentRefs = map[string]permission.Resource{
	changepreview.FieldCustomerID:        permission.ResourceCustomer,
	changepreview.FieldBillToCustomerID:  permission.ResourceCustomer,
	changepreview.FieldServiceTypeID:     permission.ResourceServiceType,
	changepreview.FieldShipmentTypeID:    permission.ResourceShipmentType,
	changepreview.FieldFormulaTemplateID: permission.ResourceFormulaTemplate,
}

var enteredShipmentLabels = map[string]string{
	changepreview.FieldCustomerID:        changepreview.LabelCustomer,
	changepreview.FieldBillToCustomerID:  "Bill to",
	changepreview.FieldServiceTypeID:     "Service type",
	changepreview.FieldShipmentTypeID:    "Shipment type",
	changepreview.FieldFormulaTemplateID: "Rating method",
	changepreview.FieldBol:               "BOL",
}

var enteredStopFields = []string{
	changepreview.FieldType,
	changepreview.FieldLocationID,
	"scheduleType",
	changepreview.FieldScheduledWindowStart,
	changepreview.FieldScheduledWindowEnd,
	changepreview.FieldPieces,
	changepreview.FieldWeight,
}

var enteredStopLabels = map[string]string{
	changepreview.FieldLocationID:           "Location",
	changepreview.FieldScheduledWindowStart: "Window opens",
	changepreview.FieldScheduledWindowEnd:   "Window closes",
}

var shipmentChargeFields = []string{"freightChargeAmount", "otherChargeAmount"}

func EnteredShipmentOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(enteredShipmentFields...),
		toolpreview.WithRefs(enteredShipmentRefs),
		toolpreview.Labels(enteredShipmentLabels),
	}
}

func EnteredStopOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(enteredStopFields...),
		toolpreview.WithRefs(map[string]permission.Resource{
			changepreview.FieldLocationID: permission.ResourceLocation,
		}),
		toolpreview.Labels(enteredStopLabels),
	}
}

func (t *CreateShipmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	entity, err := t.Draft(ctx, &params)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build("Would enter a new shipment."),
				err,
			), nil
		}

		return nil, err
	}

	plan, err := t.shipments.PreviewCreate(ctx, entity, params.Actor)
	if err != nil {
		if changepreview.IsRefusal(err) {
			return changepreview.WarnWouldFail(
				toolpreview.Build(enteredShipmentSummary(entity, nil)),
				err,
			), nil
		}

		return nil, err
	}

	changes, err := EnteredShipmentChanges(plan)
	if err != nil {
		return nil, err
	}

	preview := toolpreview.Build(enteredShipmentSummary(plan.Shipment, plan.Rating), changes...)
	warnRateCoverage(preview, plan.Advisories)

	return preview, nil
}

func warnRateCoverage(preview *agentpreview.ToolPreview, advisories []*errortypes.AdvisoryError) {
	for _, advisory := range advisories {
		if advisory == nil || advisory.RuleKey != shipment.RateCoverageRuleKey {
			continue
		}

		reason := i18n.Format(advisory.Message, advisory.Args...)
		toolpreview.Warn(preview, agentpreview.PreviewWarningRateCoverage,
			"The rate needs review: "+reason+".", reason)
	}
}

func EnteredShipmentChanges(
	plan *shipmentserviceports.ShipmentCreatePlan,
) ([]*agentpreview.RecordChange, error) {
	entity := plan.Shipment
	label := "New shipment"
	if entity.BOL != "" {
		label = "Shipment with BOL " + entity.BOL
	}

	created, err := toolpreview.Create(toolpreview.Record{
		Resource: permission.ResourceShipment,
		Label:    label,
	}, entity, EnteredShipmentOptions()...)
	if err != nil {
		return nil, err
	}
	toolpreview.AttachMoney(
		created,
		enteredShipmentMoney(entity, plan.Rating),
		toolpreview.SensitiveAs(shipmentChargeFields...),
	)

	changes := []*agentpreview.RecordChange{created}
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
			}, stop, EnteredStopOptions()...)
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
	rating *shipmentserviceports.ShipmentRatingOutcome,
) *agentpreview.MoneyPreview {
	currency := money.DefaultCurrencyCode
	if rating != nil && rating.Currency != "" {
		currency = rating.Currency
	}

	return toolpreview.MoneyBlock(
		currency,
		agentpreview.MoneyLine{Label: "Linehaul", After: entity.FreightChargeAmount},
		agentpreview.MoneyLine{Label: "Other charges", After: entity.OtherChargeAmount},
	)
}

func enteredShipmentSummary(
	entity *shipment.Shipment,
	rating *shipmentserviceports.ShipmentRatingOutcome,
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
		inflect.Pluralize("stop", "stops", stops),
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
