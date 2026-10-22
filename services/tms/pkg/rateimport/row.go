package rateimport

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/decimalutils"
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

func RateTypeTemplateNames() []string {
	names := make([]string, 0, len(basisSpellings))
	seen := make(map[string]struct{}, len(basisSpellings))
	for _, name := range basisSpellings {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

type Template struct {
	ID     pulid.ID
	Inputs []tarifftypes.InputDecl
}

type TemplateResolver func(name string) (*Template, bool)

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

	templateName, err := parseBasis(mapping.Value(row, FieldRatingBasis))
	if err != nil {
		return nil, err
	}

	template, ok := templates(templateName)
	if !ok {
		return nil, fmt.Errorf(
			"no active formula template named %q exists to price this rate type",
			templateName,
		)
	}

	rate, err := decimalutils.ParseMoneyText(mapping.Value(row, FieldRate))
	if err != nil {
		return nil, fmt.Errorf("rate: %w", err)
	}

	if !rate.Valid {
		return nil, errNoRate
	}

	inputs, err := ruleInputs(mapping, row, template, rate.Decimal)
	if err != nil {
		return nil, err
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
		Direction:             rateagreement.DirectionDirectional,
		FormulaTemplateID:     template.ID,
		Inputs:                inputs,
		Currency:              mapping.Value(row, FieldCurrency),
	}

	return rule, nil
}

var errNoRate = errors.New("this row names a lane but no rate, so it would price nothing")

var (
	errNoRateInput    = errors.New("this rate type's tariff takes no rate")
	errNoMinimumInput = errors.New("this rate type's tariff takes no minimum charge")
)

func ruleInputs(
	mapping Mapping,
	row []string,
	template *Template,
	rate decimal.Decimal,
) (tarifftypes.Inputs, error) {
	rateDecl := tarifftypes.FindInput(template.Inputs, formulatemplate.StandardInputRate)
	if rateDecl == nil {
		return nil, errNoRateInput
	}
	rateValue, err := rateDecl.ScalarValue(rate)
	if err != nil {
		return nil, fmt.Errorf("rate: %w", err)
	}
	inputs := tarifftypes.Inputs{formulatemplate.StandardInputRate: rateValue}

	minimum, err := decimalutils.ParseMoneyText(mapping.Value(row, FieldMinCharge))
	if err != nil {
		return nil, fmt.Errorf("minimum charge: %w", err)
	}
	if !minimum.Valid {
		return inputs, nil
	}
	minimumDecl := tarifftypes.FindInput(
		template.Inputs,
		formulatemplate.StandardInputMinimumCharge,
	)
	if minimumDecl == nil {
		return nil, errNoMinimumInput
	}
	minimumValue, err := minimumDecl.ScalarValue(minimum.Decimal)
	if err != nil {
		return nil, fmt.Errorf("minimum charge: %w", err)
	}
	inputs[formulatemplate.StandardInputMinimumCharge] = minimumValue

	return inputs, nil
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

func UnkeyedEnd(rule *rateagreement.RateAgreementRule) error {
	origin, destination := rule.KeyedEnds()
	switch {
	case !origin:
		return fmt.Errorf(
			"origin: %q cannot be read as a %s", rule.OriginScopeValue, rule.OriginScopeType,
		)
	case !destination:
		return fmt.Errorf(
			"destination: %q cannot be read as a %s",
			rule.DestinationScopeValue,
			rule.DestinationScopeType,
		)
	default:
		return nil
	}
}

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
