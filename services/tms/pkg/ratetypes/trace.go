package ratetypes

import "github.com/shopspring/decimal"

// The trace is the answer to "why did I get this rate".
//
// It is written on every rating, including the ones that fell through to a
// formula template and the ones that found nothing, because the cases worth
// explaining are exactly the ones nobody predicted. Everything here is a plain
// value: the trace is stored as it was computed and read back unchanged, so an
// explanation given six months later is the explanation that was true at the
// time rather than a re-derivation against data that has since moved.

// ComponentKind classifies one line of a rate's arithmetic.
type ComponentKind string

const (
	ComponentKindLinehaul           = ComponentKind("Linehaul")
	ComponentKindWeightBreak        = ComponentKind("WeightBreak")
	ComponentKindDeficitBump        = ComponentKind("DeficitBump")
	ComponentKindDiscount           = ComponentKind("Discount")
	ComponentKindAbsoluteMinCharge  = ComponentKind("AbsoluteMinCharge")
	ComponentKindMinimumCharge      = ComponentKind("MinimumCharge")
	ComponentKindMaximumCharge      = ComponentKind("MaximumCharge")
	ComponentKindDeficitDistance    = ComponentKind("DeficitDistance")
	ComponentKindAccessorial        = ComponentKind("Accessorial")
	ComponentKindFuelSurcharge      = ComponentKind("FuelSurcharge")
	ComponentKindDetention          = ComponentKind("Detention")
	ComponentKindCurrencyConversion = ComponentKind("CurrencyConversion")
	ComponentKindRounding           = ComponentKind("Rounding")
	ComponentKindManualOverride     = ComponentKind("ManualOverride")
)

func (ck ComponentKind) String() string {
	return string(ck)
}

// ComponentSource names where a component's number came from, so a reader can
// go straight to the record that produced it.
type ComponentSource string

const (
	ComponentSourceAgreementRule       = ComponentSource("AgreementRule")
	ComponentSourceRateMatrix          = ComponentSource("RateMatrix")
	ComponentSourceFormulaTemplate     = ComponentSource("FormulaTemplate")
	ComponentSourceFuelProgram         = ComponentSource("FuelProgram")
	ComponentSourceDetentionPolicy     = ComponentSource("DetentionPolicy")
	ComponentSourceAccessorialSchedule = ComponentSource("AccessorialSchedule")
	ComponentSourceExchangeRate        = ComponentSource("ExchangeRate")
	ComponentSourceManual              = ComponentSource("Manual")
)

func (cs ComponentSource) String() string {
	return string(cs)
}

// Component is one step of the arithmetic, in the order it was applied.
type Component struct {
	Sequence int16         `json:"sequence"`
	Kind     ComponentKind `json:"kind"`
	Code     string        `json:"code,omitempty"`
	Label    string        `json:"label"`
	// Basis is the calculation in words — "1,240.0 mi @ $2.15/mi". It is what
	// goes on a dispute letter, so it is stored rather than reconstructed.
	Basis        string              `json:"basis,omitempty"`
	Quantity     decimal.NullDecimal `json:"quantity,omitempty"`
	Rate         decimal.NullDecimal `json:"rate,omitempty"`
	Amount       decimal.Decimal     `json:"amount"`
	RunningTotal decimal.Decimal     `json:"runningTotal"`
	Source       ComponentSource     `json:"source"`
	SourceID     string              `json:"sourceId,omitempty"`
	SourceName   string              `json:"sourceName,omitempty"`
	// Detail carries whatever the step needs to be reproduced — the matrix cell
	// keys read, a weight break's bounds, a formula's resolved variables.
	Detail map[string]any `json:"detail,omitempty"`
}

// Candidate is one rule that was considered, whether or not it won.
//
// Losers are kept deliberately. "No rate applied" and "a rate applied but not
// the one you expected" are different problems, and only the full list tells
// them apart.
type Candidate struct {
	AgreementID       string       `json:"agreementId"`
	AgreementCode     string       `json:"agreementCode,omitempty"`
	AgreementName     string       `json:"agreementName,omitempty"`
	RuleID            string       `json:"ruleId"`
	RuleLabel         string       `json:"ruleLabel,omitempty"`
	LaneKey           string       `json:"laneKey,omitempty"`
	Won               bool         `json:"won"`
	Rank              int16        `json:"rank"`
	SpecificityScore  int32        `json:"specificityScore"`
	RulePriority      int16        `json:"rulePriority"`
	AgreementPriority int16        `json:"agreementPriority"`
	EffectiveFrom     int64        `json:"effectiveFrom"`
	EffectiveTo       *int64       `json:"effectiveTo,omitempty"`
	RejectReason      RejectReason `json:"rejectReason,omitempty"`
	RejectDetail      string       `json:"rejectDetail,omitempty"`
	// MatchedOn lists what the rule matched the shipment on, in the same
	// vocabulary mode profile provenance uses, so the two read alike.
	MatchedOn []string `json:"matchedOn,omitempty"`
	// Amount is filled in only where pricing the loser was worth the cost —
	// rate shopping, and the "what would this have charged" view.
	Amount decimal.NullDecimal `json:"amount,omitempty"`
}

// Guardrail records a bound that changed the answer.
type Guardrail struct {
	Kind    ComponentKind   `json:"kind"`
	Applied bool            `json:"applied"`
	Bound   decimal.Decimal `json:"bound"`
	Raw     decimal.Decimal `json:"raw"`
	Result  decimal.Decimal `json:"result"`
}

// FXConversion records the exchange applied, so a rate stays reproducible after
// the market has moved.
type FXConversion struct {
	FromCurrency   string          `json:"fromCurrency"`
	ToCurrency     string          `json:"toCurrency"`
	Rate           decimal.Decimal `json:"rate"`
	ExchangeRateID string          `json:"exchangeRateId,omitempty"`
	RateDate       string          `json:"rateDate,omitempty"`
}

// Totals is the money, split the way an invoice splits it.
type Totals struct {
	Linehaul    decimal.Decimal     `json:"linehaul"`
	Fuel        decimal.Decimal     `json:"fuel"`
	Accessorial decimal.Decimal     `json:"accessorial"`
	Total       decimal.Decimal     `json:"total"`
	Cost        decimal.NullDecimal `json:"cost,omitempty"`
	Margin      decimal.NullDecimal `json:"margin,omitempty"`
	MarginPct   decimal.NullDecimal `json:"marginPercent,omitempty"`
}

// Inputs is every fact the rating read, frozen.
//
// Stops get edited, mileage engines get upgraded, commodities get
// reclassified. Without the inputs as they stood, a replay months later
// answers a question nobody asked.
type Inputs struct {
	RatingDate int64  `json:"ratingDate"`
	Weekday    int    `json:"weekday"`
	PartyType  string `json:"partyType"`
	PartyID    string `json:"partyId"`
	PartyName  string `json:"partyName,omitempty"`

	OriginLocationID      string   `json:"originLocationId,omitempty"`
	OriginCity            string   `json:"originCity,omitempty"`
	OriginStateID         string   `json:"originStateId,omitempty"`
	OriginPostalCode      string   `json:"originPostalCode,omitempty"`
	OriginZoneIDs         []string `json:"originZoneIds,omitempty"`
	DestinationLocationID string   `json:"destinationLocationId,omitempty"`
	DestinationCity       string   `json:"destinationCity,omitempty"`
	DestinationStateID    string   `json:"destinationStateId,omitempty"`
	DestinationPostalCode string   `json:"destinationPostalCode,omitempty"`
	DestinationZoneIDs    []string `json:"destinationZoneIds,omitempty"`

	Distance       decimal.Decimal `json:"distance"`
	DistanceSource string          `json:"distanceSource,omitempty"`
	Weight         decimal.Decimal `json:"weight"`
	Pieces         int64           `json:"pieces"`
	LinearFeet     decimal.Decimal `json:"linearFeet"`
	CubicFeet      decimal.Decimal `json:"cubicFeet"`
	Stops          int16           `json:"stops"`

	ServiceTypeID  string   `json:"serviceTypeId,omitempty"`
	ShipmentTypeID string   `json:"shipmentTypeId,omitempty"`
	TractorTypeID  string   `json:"tractorTypeId,omitempty"`
	TrailerTypeID  string   `json:"trailerTypeId,omitempty"`
	ServiceModel   string   `json:"serviceModel,omitempty"`
	EquipmentClass string   `json:"equipmentClass,omitempty"`
	CommodityIDs   []string `json:"commodityIds,omitempty"`
	FreightClasses []string `json:"freightClasses,omitempty"`

	// FreightClassUsed and its provenance are recorded separately because a
	// class dispute is the most common LTL billing adjustment, and "we measured
	// 8.2 pcf, which classifies as 92.5" is the entire argument.
	FreightClassUsed   string              `json:"freightClassUsed,omitempty"`
	FreightClassSource string              `json:"freightClassSource,omitempty"`
	DensityPcf         decimal.NullDecimal `json:"densityPcf,omitempty"`

	HasHazmat           bool `json:"hasHazmat"`
	RequiresTempControl bool `json:"requiresTempControl"`
}

// Trace is the whole record of one rating.
type Trace struct {
	// EngineVersion is bumped whenever the scoring or pricing arithmetic
	// changes, which is what separates "the inputs moved" from "we changed how
	// we compute this" when two runs disagree.
	EngineVersion string `json:"engineVersion"`

	Inputs Inputs `json:"inputs"`

	// LaneKeysTried is what the shipment was looked up under. When nothing
	// matched, this is the answer to "why not" — the contract simply has no
	// rule keyed to any of these.
	LaneKeysTried []string `json:"laneKeysTried,omitempty"`
	// CandidateCount is how many rules the database returned, which can exceed
	// the recorded candidates when the fetch was capped.
	CandidateCount int32       `json:"candidateCount"`
	Candidates     []Candidate `json:"candidates,omitempty"`

	// TieBreak names the ordering step that actually separated the winner from
	// the runner up.
	TieBreak string `json:"tieBreak,omitempty"`

	Components []Component   `json:"components,omitempty"`
	Guardrails []Guardrail   `json:"guardrails,omitempty"`
	FX         *FXConversion `json:"fx,omitempty"`
	Totals     Totals        `json:"totals"`

	Warnings []string `json:"warnings,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// AddComponent appends a step and keeps the running total consistent, so no
// caller has to remember to.
func (t *Trace) AddComponent(component Component) {
	running := component.Amount
	if len(t.Components) > 0 {
		running = t.Components[len(t.Components)-1].RunningTotal.Add(component.Amount)
	}

	component.Sequence = int16(len(t.Components))
	component.RunningTotal = running
	t.Components = append(t.Components, component)
}

// Warn records something a reader should know that did not stop the rating —
// a missing dimension that forced a fallback, a fuel price that had to be
// carried forward.
func (t *Trace) Warn(message string) {
	t.Warnings = append(t.Warnings, message)
}

// Winner returns the candidate that priced the shipment.
func (t *Trace) Winner() *Candidate {
	for i := range t.Candidates {
		if t.Candidates[i].Won {
			return &t.Candidates[i]
		}
	}

	return nil
}
