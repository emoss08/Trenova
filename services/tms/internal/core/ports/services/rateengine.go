package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// RateShipmentRequest asks the engine to price one side of a shipment.
type RateShipmentRequest struct {
	Shipment   *shipment.Shipment
	TenantInfo pagination.TenantInfo
	PartyType  rateagreement.PartyType
	// PartyID is the carrier on a buy side rating. On the sell side it is left
	// unset and taken from the shipment's customer.
	PartyID pulid.ID
	// AsOf overrides the rating date, which is what a what-if or a replay
	// against another day's terms needs. Zero uses the shipment's own date.
	AsOf int64
	// BillingControl carries the organization's decision about what to do when
	// no agreement covers the lane.
	BillingControl *tenant.BillingControl
	// SellTotal is only read on the buy side, for carrier agreements written as
	// a share of what the customer pays.
	SellTotal decimal.NullDecimal
	// Purpose separates a rating that governs a shipment from a comparison, a
	// simulation or a standalone what-if.
	Purpose ratequote.Purpose
	// Persist writes the quote. A what-if answers the question without leaving
	// a record that competes with the shipment's real rate.
	Persist bool
	UserID  pulid.ID
}

// RatedShipment is what the engine produced.
//
// It is serialized straight to the client by the quote and explain endpoints,
// which is why it carries json tags: everything else on the wire is camel case.
type RatedShipment struct {
	// Amount is the linehaul, in the organization's billing currency. Fuel and
	// accessorials are added by the caller that owns them.
	Amount   decimal.Decimal      `json:"amount"`
	Currency string               `json:"currency"`
	Outcome  ratequote.Outcome    `json:"outcome"`
	Quote    *ratequote.RateQuote `json:"quote,omitempty"`

	// Agreement and Rule are set when a contract priced the shipment, so the
	// caller can stamp them on it without re-reading the quote.
	AgreementID *pulid.ID `json:"agreementId,omitempty"`
	RuleID      *pulid.ID `json:"ruleId,omitempty"`
	// FormulaTemplateID is set when the rule delegated to a formula, or when
	// the rating fell back to one because no agreement covered the lane.
	FormulaTemplateID *pulid.ID `json:"formulaTemplateId,omitempty"`
}

// RateEngine prices a shipment against the contracts that cover it.
type RateEngine interface {
	// RateShipment resolves and prices one side of a shipment, and records a
	// quote explaining what it did.
	RateShipment(ctx context.Context, req *RateShipmentRequest) (*RatedShipment, error)
}
