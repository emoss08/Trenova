package operationstools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var enteredShipmentFields = []string{
	common.FieldCustomerID,
	common.FieldBillToCustomerID,
	common.FieldServiceTypeID,
	common.FieldShipmentTypeID,
	common.FieldFormulaTemplateID,
	common.FieldStatus,
	common.FieldBol,
	"freightTerms",
	common.FieldPieces,
	common.FieldWeight,
	common.FieldTemperatureMin,
	common.FieldTemperatureMax,
}

var enteredShipmentRefs = map[string]permission.Resource{
	common.FieldCustomerID:        permission.ResourceCustomer,
	common.FieldBillToCustomerID:  permission.ResourceCustomer,
	common.FieldServiceTypeID:     permission.ResourceServiceType,
	common.FieldShipmentTypeID:    permission.ResourceShipmentType,
	common.FieldFormulaTemplateID: permission.ResourceFormulaTemplate,
}

var enteredShipmentLabels = map[string]string{
	common.FieldCustomerID:        common.LabelCustomer,
	common.FieldBillToCustomerID:  "Bill to",
	common.FieldServiceTypeID:     "Service type",
	common.FieldShipmentTypeID:    "Shipment type",
	common.FieldFormulaTemplateID: "Rating method",
	common.FieldBol:               "BOL",
}

var enteredStopFields = []string{
	common.FieldType,
	common.FieldLocationID,
	"scheduleType",
	common.FieldScheduledWindowStart,
	common.FieldScheduledWindowEnd,
	common.FieldPieces,
	common.FieldWeight,
}

var enteredStopLabels = map[string]string{
	common.FieldLocationID:           "Location",
	common.FieldScheduledWindowStart: "Window opens",
	common.FieldScheduledWindowEnd:   "Window closes",
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
			common.FieldLocationID: permission.ResourceLocation,
		}),
		toolpreview.Labels(enteredStopLabels),
	}
}

func (t *CreateShipmentTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	if err := common.GuardPreview(t, &params); err != nil {
		return nil, err
	}

	entity, err := t.Draft(ctx, &params)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(toolpreview.Build("Would enter a new shipment."), err), nil
		}

		return nil, err
	}

	plan, err := t.shipments.PreviewCreate(ctx, entity, params.Actor)
	if err != nil {
		if common.IsRefusal(err) {
			return common.WarnWouldFail(
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

func warnRateCoverage(preview *agent.ToolPreview, advisories []*errortypes.AdvisoryError) {
	for _, advisory := range advisories {
		if advisory == nil || advisory.RuleKey != shipment.RateCoverageRuleKey {
			continue
		}

		reason := i18n.Format(advisory.Message, advisory.Args...)
		toolpreview.Warn(preview, agent.PreviewWarningRateCoverage,
			"The rate needs review: "+reason+".", reason)
	}
}

func EnteredShipmentChanges(plan *shipmentports.ShipmentCreatePlan) ([]*agent.RecordChange, error) {
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
	rating *shipmentports.ShipmentRatingOutcome,
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
	rating *shipmentports.ShipmentRatingOutcome,
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
