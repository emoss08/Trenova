package rateimport

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// basisSpellings maps what a rate sheet says to the standard formula template
// that prices it.
//
// A sheet says "per mile" or "flat" the way a person writes it, and refusing to
// read that is how bulk import earns a reputation for being more work than
// typing the lanes in.
var basisSpellings = map[string]string{
	"flat":          formulatemplate.StandardFlatRate,
	"flatrate":      formulatemplate.StandardFlatRate,
	"permile":       formulatemplate.StandardPerMile,
	"mile":          formulatemplate.StandardPerMile,
	"mileage":       formulatemplate.StandardPerMile,
	"percwt":        formulatemplate.StandardPerCwt,
	"cwt":           formulatemplate.StandardPerCwt,
	"hundredweight": formulatemplate.StandardPerCwt,
	"perpiece":      formulatemplate.StandardPerPiece,
	"piece":         formulatemplate.StandardPerPiece,
	"perstop":       formulatemplate.StandardPerStop,
	"stop":          formulatemplate.StandardPerStop,
	"perpallet":     formulatemplate.StandardPerPallet,
	"pallet":        formulatemplate.StandardPerPallet,
	"perpound":      formulatemplate.StandardPerPound,
	"pound":         formulatemplate.StandardPerPound,
	"perlinearfoot": formulatemplate.StandardPerLinearFoot,
	"linearfoot":    formulatemplate.StandardPerLinearFoot,
	"perhour":       formulatemplate.StandardPerHour,
	"hour":          formulatemplate.StandardPerHour,
	"hourly":        formulatemplate.StandardPerHour,
	"percent":       formulatemplate.StandardPercentOfSellRate,
}

// TemplateResolver finds the organization's active formula template with a
// given name. ParseRow is pure and cannot ask the database, so the lookup is
// handed in by whoever staged the sheet.
type TemplateResolver func(name string) (pulid.ID, bool)

var scopeSpellings = map[string]rategeo.ScopeType{
	"state":      rategeo.ScopeTypeState,
	"st":         rategeo.ScopeTypeState,
	"province":   rategeo.ScopeTypeState,
	"zone":       rategeo.ScopeTypeZone,
	"region":     rategeo.ScopeTypeZone,
	"zip3":       rategeo.ScopeTypeZip3,
	"zip":        rategeo.ScopeTypeZip3,
	"postal3":    rategeo.ScopeTypeZip3,
	"zip5":       rategeo.ScopeTypeZip5,
	"postalcode": rategeo.ScopeTypeZip5,
	"city":       rategeo.ScopeTypeCityState,
	"citystate":  rategeo.ScopeTypeCityState,
	"country":    rategeo.ScopeTypeCountry,
	"location":   rategeo.ScopeTypeLocation,
	"any":        rategeo.ScopeTypeAny,
	"all":        rategeo.ScopeTypeAny,
}

// ErrBlankRow marks a row with nothing in it.
//
// Trailing blank rows are what a spreadsheet produces, and stopping an import
// over one would be absurd — but a caller has to be able to tell "nothing here"
// apart from "this failed", which a bare nil could not say.
var ErrBlankRow = errors.New("this row is empty")

// ParseRow turns one row of a rate sheet into a rule.
//
// A blank row comes back as ErrBlankRow, which callers skip. Anything else that
// cannot be read is refused with the reason, which becomes the row's error in
// the staged import.
func ParseRow(
	mapping Mapping,
	row []string,
	templates TemplateResolver,
) (*rateagreement.RateAgreementRule, error) {
	if blank(row) {
		return nil, ErrBlankRow
	}

	origin, err := parseScope(mapping, row, FieldOriginScope, FieldOriginValue, FieldOriginCity)
	if err != nil {
		return nil, fmt.Errorf("origin: %w", err)
	}

	destination, err := parseScope(
		mapping, row,
		FieldDestinationScope, FieldDestinationValue, FieldDestinationCity,
	)
	if err != nil {
		return nil, fmt.Errorf("destination: %w", err)
	}

	originKey, ok := origin.Key()
	if !ok {
		return nil, fmt.Errorf("origin: %q cannot be read as a %s", origin.Value, origin.Type)
	}

	destinationKey, ok := destination.Key()
	if !ok {
		return nil, fmt.Errorf(
			"destination: %q cannot be read as a %s", destination.Value, destination.Type,
		)
	}

	templateName, err := parseBasis(mapping.Value(row, FieldRatingBasis))
	if err != nil {
		return nil, err
	}

	templateID, ok := templates(templateName)
	if !ok {
		return nil, fmt.Errorf(
			"no active formula template named %q exists to price this rate type",
			templateName,
		)
	}

	rate, err := parseMoney(mapping.Value(row, FieldRate))
	if err != nil {
		return nil, fmt.Errorf("rate: %w", err)
	}

	if !rate.Valid {
		return nil, errNoRate
	}

	rule := &rateagreement.RateAgreementRule{
		Label:                 label(mapping, row, origin, destination),
		Status:                rateagreement.RuleStatusActive,
		OriginScopeType:       origin.Type,
		OriginScopeValue:      origin.Value,
		OriginCity:            origin.City,
		DestinationScopeType:  destination.Type,
		DestinationScopeValue: destination.Value,
		DestinationCity:       destination.City,
		LaneKey:               rategeo.LaneKey(originKey, destinationKey),
		Direction:             rateagreement.DirectionDirectional,
		FormulaTemplateID:     &templateID,
		Rate:                  rate,
		Currency:              mapping.Value(row, FieldCurrency),
	}

	if err = applyGuardrails(mapping, row, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

var errNoRate = errors.New("this row names a lane but no rate, so it would price nothing")

func applyGuardrails(
	mapping Mapping,
	row []string,
	rule *rateagreement.RateAgreementRule,
) error {
	for _, guardrail := range []struct {
		field Field
		name  string
		set   func(decimal.NullDecimal)
	}{
		{FieldMinCharge, "minimum charge", func(v decimal.NullDecimal) { rule.MinCharge = v }},
		{FieldMaxCharge, "maximum charge", func(v decimal.NullDecimal) { rule.MaxCharge = v }},
		{FieldDiscountPercent, "discount", func(v decimal.NullDecimal) { rule.DiscountPercent = v }},
	} {
		value, err := parseMoney(mapping.Value(row, guardrail.field))
		if err != nil {
			return fmt.Errorf("%s: %w", guardrail.name, err)
		}

		guardrail.set(value)
	}

	return nil
}

func blank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}

	return true
}

// parseScope reads one end of the lane.
//
// A sheet that names the kind of place wins, because it knows something the
// importer cannot infer: "606" is a postal prefix, not a state. A sheet that
// does not say is read as states, which is what most rate sheets carry.
func parseScope(
	mapping Mapping,
	row []string,
	scopeField, valueField, cityField Field,
) (rategeo.Scope, error) {
	value := mapping.Value(row, valueField)
	city := mapping.Value(row, cityField)

	scopeType := rategeo.ScopeTypeState
	if city != "" {
		scopeType = rategeo.ScopeTypeCityState
	}

	if named := mapping.Value(row, scopeField); named != "" {
		parsed, ok := scopeSpellings[normalize(named)]
		if !ok {
			return rategeo.Scope{}, fmt.Errorf("%q is not a kind of place", named)
		}

		scopeType = parsed
	}

	if value == "" && scopeType != rategeo.ScopeTypeAny {
		return rategeo.Scope{}, errHalfALane
	}

	return rategeo.Scope{Type: scopeType, Value: value, City: city}, nil
}

var errHalfALane = errors.New("this row does not say where the lane runs")

func parseBasis(raw string) (string, error) {
	// A sheet with no basis column is a flat rate sheet, which most are.
	if raw == "" {
		return formulatemplate.StandardFlatRate, nil
	}

	templateName, ok := basisSpellings[normalize(raw)]
	if !ok {
		// Guessing would be worse than refusing: the same number priced per
		// mile and priced flat are very different money.
		return "", fmt.Errorf("%q is not a rate type this importer recognises", raw)
	}

	return templateName, nil
}

// parseMoney reads a number the way a spreadsheet writes one.
//
// Currency symbols, thousands separators and trailing percent signs are what a
// person types; refusing them would send somebody back to reformat a file that
// reads perfectly well.
func parseMoney(raw string) (decimal.NullDecimal, error) {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case ',', '$', '%', ' ', ' ':
			return -1
		default:
			return r
		}
	}, raw)

	if cleaned == "" {
		return decimal.NullDecimal{}, nil
	}

	value, err := decimal.NewFromString(cleaned)
	if err != nil {
		return decimal.NullDecimal{}, fmt.Errorf("%q is not a number", raw)
	}

	return decimal.NewNullDecimal(value), nil
}

// label names the lane when the sheet does not, so a lane list stays scannable.
func label(mapping Mapping, row []string, origin, destination rategeo.Scope) string {
	if named := mapping.Value(row, FieldLabel); named != "" {
		return named
	}

	return describe(origin) + " → " + describe(destination)
}

func describe(scope rategeo.Scope) string {
	if scope.City != "" {
		return scope.City + ", " + scope.Value
	}

	if scope.Value == "" {
		return "Anywhere"
	}

	return scope.Value
}
