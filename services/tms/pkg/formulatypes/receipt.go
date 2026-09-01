package formulatypes

import "github.com/shopspring/decimal"

// ValueSource says where a value in the evaluation environment came from,
// which is the difference between "the shipment weighs this" and "somebody
// typed this into the preview".
type ValueSource string

const (
	// ValueSourceField is a value read from the record's own fields.
	ValueSourceField ValueSource = "field"
	// ValueSourceComputed is a value the schema derives from the record.
	ValueSourceComputed ValueSource = "computed"
	// ValueSourceInput is a value the caller supplied for this evaluation.
	ValueSourceInput ValueSource = "input"
	// ValueSourceOverride is a value the rating engine bound over a field,
	// such as a matrix cell standing in as baseRate.
	ValueSourceOverride ValueSource = "override"
	// ValueSourceDefault is a template variable's declared default.
	ValueSourceDefault ValueSource = "default"
	// ValueSourceSample is a value from a synthetic environment: a preview
	// or a saved scenario rather than a real record.
	ValueSourceSample ValueSource = "sample"
)

type VariableProvenance struct {
	Name   string      `json:"name"`
	Value  any         `json:"value"`
	Source ValueSource `json:"source"`
}

// LookupMatch describes which entry answered a rate-table lookup: the exact
// key that matched, or the band the key fell into.
type LookupMatch struct {
	MatchedKey string           `json:"matchedKey,omitempty"`
	BandMin    *decimal.Decimal `json:"bandMin,omitempty"`
	BandMax    *decimal.Decimal `json:"bandMax,omitempty"`
}

type LookupTrace struct {
	// Scope is "expression" for the main formula or the breakdown line's name.
	Scope string       `json:"scope"`
	Table string       `json:"table"`
	Keys  []any        `json:"keys"`
	Value float64      `json:"value"`
	Match *LookupMatch `json:"match,omitempty"`
	Error string       `json:"error,omitempty"`
}

// Receipt is everything needed to read a formula result the way a person
// would: what each variable was and where it came from, which table rows were
// consulted, the amount before guardrails and rounding, and which version of
// the template did the arithmetic.
type Receipt struct {
	Variables      []VariableProvenance `json:"variables"`
	Lookups        []LookupTrace        `json:"lookups,omitempty"`
	RawAmount      decimal.Decimal      `json:"rawAmount"`
	VersionNumber  int64                `json:"versionNumber,omitempty"`
	EffectiveFrom  *int64               `json:"effectiveFrom,omitempty"`
	DurationMicros int64                `json:"durationMicros,omitempty"`
}

// VariableMap flattens the provenance list back to name → value.
func (r *Receipt) VariableMap() map[string]any {
	if r == nil {
		return map[string]any{}
	}
	values := make(map[string]any, len(r.Variables))
	for _, variable := range r.Variables {
		values[variable.Name] = variable.Value
	}
	return values
}

// WithoutTiming returns a copy suitable for persisting or comparing: the
// wall-clock duration is dropped, since it differs on every run.
func (r *Receipt) WithoutTiming() *Receipt {
	if r == nil {
		return nil
	}
	copied := *r
	copied.DurationMicros = 0
	return &copied
}
