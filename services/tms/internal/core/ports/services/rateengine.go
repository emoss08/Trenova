package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
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
	// FormulaOnly prices the shipment through the formula template it already
	// carries, without resolving an agreement for it.
	//
	// This is the routine recalculation: a stop moves, a commodity is added,
	// the mileage changes, and the linehaul has to follow. A contract is
	// applied to a shipment once, deliberately, and after that the shipment's
	// own rating fields are what price it — so re-resolving the contract on
	// every save would silently replace whatever a rater had since agreed with
	// the customer. It also skips the zone and lane lookups, which are the only
	// queries a recalculation would otherwise make.
	FormulaOnly bool
	UserID      pulid.ID

	// SimulateAgreementID lets one agreement price this shipment whatever its
	// status. It is what a simulation of a draft contract needs, and it must
	// never be set on a rating that governs a shipment: an unsigned contract
	// would be invoicing.
	SimulateAgreementID *pulid.ID
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
	// BaseRate is the per-unit rate the template was priced with — the lane
	// rate on the rule, or the value in the matrix cell that matched.
	//
	// It is what lets a contract's answer be seated on a shipment as its own
	// rating method plus its own base rate, rather than as a total nobody can
	// re-derive. A rule that binds no rate leaves this unset, and the
	// shipment's existing base rate stands.
	BaseRate decimal.NullDecimal `json:"baseRate,omitempty"`
}

// ShopStrategy is what "best" means for one shopping run.
//
// It is a per-request choice rather than a setting, because the answer changes
// with the load: a cheap carrier is the right pick on a lane with room in it and
// the wrong pick on one the customer will cancel over.
type ShopStrategy string

const (
	// ShopStrategyLeastCost ranks by what the carrier charges, cheapest first.
	ShopStrategyLeastCost = ShopStrategy("LeastCost")
	// ShopStrategyBestMargin ranks by what is left after paying them, which is
	// not the same order as least cost once contracts price accessorials and
	// fuel differently.
	ShopStrategyBestMargin = ShopStrategy("BestMargin")
	// ShopStrategyGuideRank keeps the routing guide's own order, so a committed
	// primary carrier is offered the load first whatever the spot market says.
	ShopStrategyGuideRank = ShopStrategy("GuideRank")
	// ShopStrategyFastestAccept ranks by the shortest offer expiry the guide
	// allows, for a load that has to move now.
	ShopStrategyFastestAccept = ShopStrategy("FastestAccept")
)

func (s ShopStrategy) IsValid() bool {
	switch s {
	case ShopStrategyLeastCost,
		ShopStrategyBestMargin,
		ShopStrategyGuideRank,
		ShopStrategyFastestAccept:
		return true
	default:
		return false
	}
}

func (s ShopStrategy) String() string { return string(s) }

// ShopRequest asks what each carrier would charge to haul this shipment.
type ShopRequest struct {
	Shipment   *shipment.Shipment
	TenantInfo pagination.TenantInfo
	Strategy   ShopStrategy

	// CarrierIDs is an explicit shortlist. Left empty, the shipment's routing
	// guide supplies the candidates, which is the ordinary case.
	CarrierIDs []pulid.ID

	// AsOf overrides the rating date. Zero uses the shipment's own.
	AsOf int64

	// BillingControl decides what a lane no carrier contract covers does.
	BillingControl *tenant.BillingControl

	// SellTotal is what the load is being sold for. It is what makes margin
	// answerable, and carrier contracts written as a share of the sell price
	// cannot be priced without it.
	SellTotal decimal.NullDecimal

	// Limit caps the options returned. Zero means the default.
	Limit int

	// Persist writes each option's quote. Shopping to decide is worth
	// recording; shopping to look around is not.
	Persist bool
	UserID  pulid.ID
}

// ShopOption is one carrier's answer.
type ShopOption struct {
	CarrierID   pulid.ID `json:"carrierId"`
	CarrierName string   `json:"carrierName"`

	// Rank is this option's position in the returned order, from one.
	Rank int `json:"rank"`
	// GuideRank is the routing guide's own rank, zero when the carrier did not
	// come from a guide.
	GuideRank int16 `json:"guideRank"`
	// OfferTTLSeconds is how long the guide gives the carrier to accept.
	OfferTTLSeconds int32 `json:"offerTtlSeconds"`

	Outcome  ratequote.Outcome `json:"outcome"`
	Cost     decimal.Decimal   `json:"cost"`
	Currency string            `json:"currency"`

	Margin ratetypes.MarginVerdict `json:"margin"`

	Quote       *ratequote.RateQuote `json:"quote,omitempty"`
	AgreementID *pulid.ID            `json:"agreementId,omitempty"`
	RuleID      *pulid.ID            `json:"ruleId,omitempty"`

	// Note says why this option ranked where it did, or why it could not be
	// priced at all. It is what makes a shopping result answerable months later.
	Note string `json:"note,omitempty"`
}

// Priced reports whether a contract actually put a number on this carrier.
func (o *ShopOption) Priced() bool {
	return o.Outcome.Priced()
}

// ShopResult is the ranked answer, plus what it was ranked by.
type ShopResult struct {
	Strategy ShopStrategy  `json:"strategy"`
	Options  []*ShopOption `json:"options"`

	// RoutingGuideID names the guide the candidates came from, absent when the
	// caller supplied its own shortlist or no guide covered the lane.
	RoutingGuideID *pulid.ID `json:"routingGuideId,omitempty"`

	// SellTotal is the number margin was measured against, echoed back so a
	// stored result explains itself.
	SellTotal decimal.NullDecimal `json:"sellTotal"`

	// Warnings covers what the caller should know but that did not stop the
	// shopping: carriers with no contract, a guide that matched nothing.
	Warnings []string `json:"warnings,omitempty"`
}

// RateEngine prices a shipment against the contracts that cover it.
type RateEngine interface {
	// RateShipment resolves and prices one side of a shipment, and records a
	// quote explaining what it did.
	RateShipment(ctx context.Context, req *RateShipmentRequest) (*RatedShipment, error)

	// Shop prices the same shipment against several carriers and ranks them.
	Shop(ctx context.Context, req *ShopRequest) (*ShopResult, error)
}
