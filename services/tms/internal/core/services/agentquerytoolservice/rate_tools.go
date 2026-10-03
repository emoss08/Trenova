package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

/*
"Why is this shipment priced at $2,840?"

The rate engine already answers that: every rating writes a trace — which rules
were considered, which one won and on what tie-break, every component with its
basis and running total, the guardrails that bit, and the warnings. Nobody
could read it, because it lived in a JSONB column behind a quote id.

This reads the trace of the quote that is actually governing the shipment,
rather than re-rating. A fresh rating months later answers a question nobody
asked: contracts get amended, fuel indexes move, stops get edited. The number
on the invoice came from the stored trace, so the explanation has to come from
the same place or it is an explanation of something else.
*/

// quoteReader is the slice of the quote repository this tool uses.
type quoteReader interface {
	GetAppliedForShipment(
		ctx context.Context,
		req *repositories.GetShipmentRateQuoteRequest,
	) (*ratequote.RateQuote, error)
}

// shipmentRatingReader reads the rating a shipment carries itself, for the
// shipments no agreement priced.
type shipmentRatingReader interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
}

type explainRateTool struct {
	quotes    quoteReader
	shipments shipmentRatingReader
}

func newExplainRateTool(
	quotes repositories.RateQuoteRepository,
	shipments repositories.ShipmentRepository,
) serviceports.AgentQueryTool {
	return &explainRateTool{quotes: quotes, shipments: shipments}
}

func (t *explainRateTool) Name() string { return "explain_rate" }

func (t *explainRateTool) Description() string {
	return "Explain how a saved shipment's rate was arrived at: the agreement and rule that " +
		"priced it and every charge with its arithmetic. It also shows what separated that " +
		"rule from the others, a running total, any floor or ceiling that changed the " +
		"number, and anything the engine warned about. Use it for \"why is this priced at X\", for a " +
		"customer disputing a charge, and before quoting a similar lane. It reads the " +
		"rating that actually produced the shipment's price, not a fresh one, so the " +
		"figures match the invoice even after the contract has since changed."
}

func (t *explainRateTool) ParamSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"shipmentId"},
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment whose rate to explain, from search_shipments " +
					"or list_shipments, or the page you are on.",
			},
			"side": agenttoolschema.Enum(
				"Which side of the shipment to price: what the customer is "+
					"charged, or what the carrier is paid. Defaults to Customer.",
				ratingSides,
			),
		},
		"additionalProperties": false,
	}
}

func (t *explainRateTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceRateMatrix,
	})
}

type rateExplanation struct {
	ShipmentID string `json:"shipmentId"`
	Side       string `json:"side"`
	Currency   string `json:"currency,omitempty"`
	// Winner names what priced the shipment, and TieBreak why it rather than
	// the runner-up — which is the first question a disputed rate raises.
	Winner   *rateWinnerRow  `json:"winner,omitempty"`
	TieBreak string          `json:"tieBreak,omitempty"`
	Rejected []rateRejectRow `json:"rejected,omitempty"`
	// LaneKeysTried answers "why did nothing apply": the contract has no rule
	// keyed to any of these.
	LaneKeysTried []string           `json:"laneKeysTried,omitempty"`
	Components    []rateComponentRow `json:"components,omitempty"`
	Guardrails    []rateGuardrailRow `json:"guardrails,omitempty"`
	Totals        rateTotalsRow      `json:"totals"`
	Warnings      []string           `json:"warnings,omitempty"`
	Error         string             `json:"error,omitempty"`
	EngineVersion string             `json:"engineVersion,omitempty"`
	// PricedBy is how a shipment no agreement priced got its price: the
	// formula template it was rated with, or the amount someone entered.
	PricedBy *pricedByRow `json:"pricedBy,omitempty"`
	// Note is set when there is nothing to explain, so the model says that
	// rather than reporting a zero as a price.
	Note string `json:"note,omitempty"`
}

// pricedByRow is the rating a shipment stores on itself: no agreement and no
// quote trace, but the method, the expression and what it came to.
type pricedByRow struct {
	// Method is the formula template that priced it, "Flat Rate" say.
	Method      string           `json:"method,omitempty"`
	Source      string           `json:"source,omitempty"`
	Explanation string           `json:"explanation,omitempty"`
	Expression  string           `json:"expression,omitempty"`
	Variables   map[string]any   `json:"variables,omitempty"`
	BaseRate    *decimal.Decimal `json:"baseRate,omitempty"`
	RatedAt     int64            `json:"ratedAt,omitempty"`
	// Override is an amount a person entered over the rating, and why.
	Override       *decimal.Decimal `json:"override,omitempty"`
	OverrideReason string           `json:"overrideReason,omitempty"`
	Locked         bool             `json:"locked,omitempty"`
}

type rateWinnerRow struct {
	AgreementCode string   `json:"agreementCode,omitempty"`
	AgreementName string   `json:"agreementName,omitempty"`
	RuleLabel     string   `json:"ruleLabel,omitempty"`
	LaneKey       string   `json:"laneKey,omitempty"`
	MatchedOn     []string `json:"matchedOn,omitempty"`
}

type rateRejectRow struct {
	AgreementCode string `json:"agreementCode,omitempty"`
	RuleLabel     string `json:"ruleLabel,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Detail        string `json:"detail,omitempty"`
}

type rateComponentRow struct {
	Label string `json:"label"`
	Kind  string `json:"kind,omitempty"`
	Code  string `json:"code,omitempty"`
	// Basis is the calculation in words — "1,240.0 mi @ $2.15/mi". It is what
	// goes on a dispute letter, so it is passed through rather than rebuilt.
	Basis        string          `json:"basis,omitempty"`
	Amount       decimal.Decimal `json:"amount"`
	RunningTotal decimal.Decimal `json:"runningTotal"`
	Source       string          `json:"source,omitempty"`
	SourceName   string          `json:"sourceName,omitempty"`
}

// rateGuardrailRow is a floor or ceiling. Only the ones that actually changed
// the number are reported: a limit that did not bite is not an explanation.
type rateGuardrailRow struct {
	Kind   string          `json:"kind"`
	Bound  decimal.Decimal `json:"bound"`
	Raw    decimal.Decimal `json:"raw"`
	Result decimal.Decimal `json:"result"`
}

type rateTotalsRow struct {
	Linehaul    decimal.Decimal  `json:"linehaul"`
	Fuel        decimal.Decimal  `json:"fuel"`
	Accessorial decimal.Decimal  `json:"accessorial"`
	Total       decimal.Decimal  `json:"total"`
	Margin      *decimal.Decimal `json:"margin,omitempty"`
	MarginPct   *decimal.Decimal `json:"marginPercent,omitempty"`
}

func (t *explainRateTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	shipmentID, err := pulid.Parse(optionalString(params.Params, "shipmentId"))
	if err != nil {
		return nil, fmt.Errorf(
			"%q is not a shipment id",
			optionalString(params.Params, "shipmentId"),
		)
	}

	side := rateagreement.PartyTypeCustomer
	if optionalString(params.Params, "side") == "Carrier" {
		side = rateagreement.PartyTypeCarrier
	}

	quote, err := t.quotes.GetAppliedForShipment(ctx, &repositories.GetShipmentRateQuoteRequest{
		ShipmentID: shipmentID,
		PartyType:  side,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
	})
	if err != nil {
		return nil, err
	}

	explanation := rateExplanation{ShipmentID: shipmentID.String(), Side: string(side)}
	if (quote == nil || quote.Trace == nil) && side == rateagreement.PartyTypeCustomer &&
		t.shipments != nil {
		record, err := t.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
			ID: shipmentID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
		})
		if err != nil {
			return nil, err
		}
		if stored, ok := explainStored(explanation, record); ok {
			return stored, nil
		}
	}
	if quote == nil || quote.Trace == nil {
		explanation.Note = "This shipment has no rating on record for that side, so there " +
			"is no price to explain. It may be rated manually, or not yet rated."

		return explanation, nil
	}

	return explainTrace(explanation, quote.Trace), nil
}

func explainTrace(into rateExplanation, trace *ratetypes.Trace) rateExplanation {
	into.EngineVersion = trace.EngineVersion
	into.TieBreak = trace.TieBreak
	into.LaneKeysTried = trace.LaneKeysTried
	into.Warnings = trace.Warnings
	into.Error = trace.Error
	into.Totals = totalsRow(trace.Totals)
	if trace.FX != nil {
		into.Currency = trace.FX.ToCurrency
	}

	if winner := trace.Winner(); winner != nil {
		into.Winner = &rateWinnerRow{
			AgreementCode: winner.AgreementCode,
			AgreementName: winner.AgreementName,
			RuleLabel:     winner.RuleLabel,
			LaneKey:       winner.LaneKey,
			MatchedOn:     winner.MatchedOn,
		}
	}

	// The losers are the other half of the answer: "no rate applied" and "a
	// rate applied but not the one you expected" are different problems, and
	// the reason a rule was passed over is what tells them apart.
	for i := range trace.Candidates {
		candidate := &trace.Candidates[i]
		if candidate.Won || candidate.RejectReason == "" {
			continue
		}
		into.Rejected = append(into.Rejected, rateRejectRow{
			AgreementCode: candidate.AgreementCode,
			RuleLabel:     candidate.RuleLabel,
			Reason:        string(candidate.RejectReason),
			Detail:        candidate.RejectDetail,
		})
	}

	into.Components = make([]rateComponentRow, 0, len(trace.Components))
	for i := range trace.Components {
		component := &trace.Components[i]
		into.Components = append(into.Components, rateComponentRow{
			Label:        component.Label,
			Kind:         string(component.Kind),
			Code:         component.Code,
			Basis:        component.Basis,
			Amount:       component.Amount,
			RunningTotal: component.RunningTotal,
			Source:       string(component.Source),
			SourceName:   component.SourceName,
		})
	}

	for i := range trace.Guardrails {
		guardrail := &trace.Guardrails[i]
		if !guardrail.Applied {
			continue
		}
		into.Guardrails = append(into.Guardrails, rateGuardrailRow{
			Kind:   string(guardrail.Kind),
			Bound:  guardrail.Bound,
			Raw:    guardrail.Raw,
			Result: guardrail.Result,
		})
	}

	return into
}

func totalsRow(totals ratetypes.Totals) rateTotalsRow {
	row := rateTotalsRow{
		Linehaul:    totals.Linehaul,
		Fuel:        totals.Fuel,
		Accessorial: totals.Accessorial,
		Total:       totals.Total,
	}
	if totals.Margin.Valid {
		row.Margin = &totals.Margin.Decimal
	}
	if totals.MarginPct.Valid {
		row.MarginPct = &totals.MarginPct.Decimal
	}

	return row
}

// explainStored explains a shipment from the rating it carries, when no
// agreement priced it. A spot load priced from its formula template has no
// quote trace, and "no rating on record" was the wrong answer for it: the
// record says which template priced it, with what expression, and what it
// came to. Nothing is reported when the shipment carries no rating and no
// amount either.
func explainStored(into rateExplanation, record *shipment.Shipment) (rateExplanation, bool) {
	if record == nil {
		return into, false
	}
	detail := record.RatingDetail
	override := record.RateOverrideAmount
	if detail == nil && !override.Valid {
		return into, false
	}

	priced := &pricedByRow{
		OverrideReason: record.RateOverrideReason,
		Locked:         record.RateLocked,
	}
	if record.BaseRate.Valid && !record.BaseRate.Decimal.IsZero() {
		base := record.BaseRate.Decimal
		priced.BaseRate = &base
	}
	if override.Valid {
		amount := override.Decimal
		priced.Override = &amount
	}

	var running decimal.Decimal
	if detail != nil {
		priced.Method = detail.FormulaTemplateName
		priced.Source = detail.Source
		priced.Explanation = detail.Explanation
		priced.Expression = detail.Expression
		priced.Variables = detail.ResolvedVariables
		priced.RatedAt = detail.RatedAt

		into.Components = make([]rateComponentRow, 0, len(detail.Breakdown))
		for _, item := range detail.Breakdown {
			amount := decimal.NewFromFloat(item.Amount)
			running = running.Add(amount)
			label := item.Label
			if label == "" {
				label = item.Name
			}
			row := rateComponentRow{
				Label:        label,
				Amount:       amount,
				RunningTotal: running,
				Source:       "Formula",
				SourceName:   detail.FormulaTemplateName,
			}
			if len(detail.Breakdown) == 1 {
				row.Basis = detail.Expression
			}
			into.Components = append(into.Components, row)
		}
	}

	freight := running
	if record.FreightChargeAmount.Valid && !record.FreightChargeAmount.Decimal.IsZero() {
		freight = record.FreightChargeAmount.Decimal
	}
	into.Totals.Linehaul = freight
	into.Totals.Total = freight
	if record.TotalChargeAmount.Valid && !record.TotalChargeAmount.Decimal.IsZero() {
		into.Totals.Total = record.TotalChargeAmount.Decimal
		into.Totals.Accessorial = into.Totals.Total.Sub(freight)
	}
	into.PricedBy = priced
	into.Note = "No rate agreement priced this shipment, so there is no agreement to explain. " +
		"pricedBy is how it was priced instead: the formula template named in method, the " +
		"expression it evaluated, and any amount a person entered over it."

	return into, true
}
