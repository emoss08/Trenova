// Package rateimport turns a rate sheet into rules, and says what committing it
// would change.
//
// Everything here is pure. A rate sheet is somebody's spreadsheet, and the two
// things that go wrong with one — a column read as the wrong field, and a commit
// that changes more than its author expected — are both decided before anything
// touches the database. Keeping that decision testable is the point.
package rateimport

import (
	"errors"
	"strings"
	"unicode"
)

// Field is one column a rate sheet can supply.
type Field string

const (
	FieldLabel Field = "label"

	FieldOriginScope      Field = "originScope"
	FieldOriginValue      Field = "originValue"
	FieldOriginCity       Field = "originCity"
	FieldDestinationScope Field = "destinationScope"
	FieldDestinationValue Field = "destinationValue"
	FieldDestinationCity  Field = "destinationCity"

	FieldRatingBasis Field = "ratingBasis"
	FieldRate        Field = "rate"
	FieldCurrency    Field = "currency"

	FieldMinCharge       Field = "minCharge"
	FieldMaxCharge       Field = "maxCharge"
	FieldDiscountPercent Field = "discountPercent"

	FieldEffectiveFrom Field = "effectiveFrom"
	FieldEffectiveTo   Field = "effectiveTo"

	FieldServiceType   Field = "serviceType"
	FieldEquipmentType Field = "equipmentType"
	FieldMinWeight     Field = "minWeight"
	FieldMaxWeight     Field = "maxWeight"
)

// synonyms are the header names each field answers to, normalized.
//
// The order within a field does not matter; the order of the fields does not
// either, because a header matches at most one. What matters is that a longer,
// more specific phrase is not shadowed by a shorter one — "rate type" must not
// be read as "rate" — which is why matching is exact against this table rather
// than a prefix or substring test.
var synonyms = map[Field][]string{
	FieldLabel: {"label", "lane", "lanename", "description", "name"},

	FieldOriginScope: {"originscope", "origintype", "originlevel", "origeo", "originkind"},
	FieldOriginValue: {
		"origin", "originstate", "originzone", "originzip", "originpostal",
		"originzip3", "originregion", "from", "fromstate", "origincode",
	},
	FieldOriginCity: {"origincity", "fromcity"},

	FieldDestinationScope: {
		"destinationscope", "destinationtype", "deststype", "desttype",
		"destinationlevel", "destinationkind",
	},
	FieldDestinationValue: {
		"destination", "destinationstate", "deststate", "destinationzone",
		"destzone", "destinationzip", "destzip", "destinationpostal",
		"destinationregion", "to", "tostate", "destinationcode", "dest",
	},
	FieldDestinationCity: {"destinationcity", "destcity", "tocity"},

	FieldRatingBasis: {"ratingbasis", "ratetype", "basis", "ratemethod", "pricingmethod"},
	FieldRate: {
		"rate", "amount", "price", "linehaul", "linehaulrate", "baserate", "charge",
	},
	FieldCurrency: {"currency", "curr", "ccy"},

	FieldMinCharge:       {"mincharge", "minimum", "minimumcharge", "min", "floor"},
	FieldMaxCharge:       {"maxcharge", "maximum", "maximumcharge", "max", "ceiling", "cap"},
	FieldDiscountPercent: {"discount", "discountpercent", "discountpct", "off"},

	FieldEffectiveFrom: {
		"effectivefrom", "effective", "effectivedate", "startdate", "start",
		"validfrom", "from_date",
	},
	FieldEffectiveTo: {
		"effectiveto", "expires", "expirydate", "expirationdate", "enddate",
		"end", "validto", "through",
	},

	FieldServiceType:   {"servicetype", "service", "serviceleveltype", "servicelevel"},
	FieldEquipmentType: {"equipmenttype", "equipment", "trailertype", "equip"},
	FieldMinWeight:     {"minweight", "weightfrom", "fromweight", "weightmin"},
	FieldMaxWeight:     {"maxweight", "weightto", "toweight", "weightmax"},
}

// lookup inverts the synonym table once, at start.
var lookup = buildLookup()

func buildLookup() map[string]Field {
	table := make(map[string]Field, 128)

	for field, names := range synonyms {
		for _, name := range names {
			table[name] = field
		}
	}

	return table
}

// Mapping says which column supplies which field.
type Mapping map[Field]int

// GuessMapping matches a sheet's headers to the fields a rule needs, and
// reports the ones it could not place.
//
// An unplaced header is reported rather than dropped: silently ignoring a
// column is how a sheet imports looking complete while a discount nobody
// noticed never made it in.
//
// Two columns claiming the same field is ambiguous, and picking between them
// would price a tariff off the wrong column. The first wins and the second is
// reported, so somebody decides rather than the importer.
func GuessMapping(headers []string) (mapping Mapping, unplaced []string) {
	mapping = make(Mapping, len(headers))
	unplaced = make([]string, 0)

	for index, header := range headers {
		normalized := normalize(header)
		if normalized == "" {
			// Padding, not a field somebody forgot to name.
			continue
		}

		field, known := lookup[normalized]
		if !known {
			unplaced = append(unplaced, strings.TrimSpace(header))
			continue
		}

		if _, taken := mapping[field]; taken {
			unplaced = append(unplaced, strings.TrimSpace(header))
			continue
		}

		mapping[field] = index
	}

	return mapping, unplaced
}

// normalize strips everything about a header that carries no meaning: case, and
// whatever spacing and punctuation the system that produced the sheet used.
func normalize(header string) string {
	var b strings.Builder

	for _, r := range header {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}

	return b.String()
}

// Has reports whether a field was mapped to a column.
func (m Mapping) Has(field Field) bool {
	_, ok := m[field]

	return ok
}

var (
	// ErrNoRateColumn means nothing in the sheet says what to charge.
	ErrNoRateColumn = errors.New("this sheet has no rate column, so nothing in it prices anything")
	// ErrNoOriginColumn means the sheet only describes one end of the lane.
	ErrNoOriginColumn = errors.New("this sheet does not say where lanes start")
	// ErrNoDestinationColumn is the same for the other end.
	ErrNoDestinationColumn = errors.New("this sheet does not say where lanes end")
)

// Validate reports whether a mapping can produce rules at all.
//
// It is a different question from whether every column was placed: a sheet can
// carry a sales rep column nobody needs and still import perfectly well. What
// it cannot do is describe half a lane, or omit the price.
//
// Every missing column is reported at once. Reporting one, waiting for a
// re-upload, then reporting the next turns fixing a template into as many
// round trips as it has mistakes.
func (m Mapping) Validate() error {
	return errors.Join(m.Problems()...)
}

// Problems lists everything that stops this mapping from producing rules.
func (m Mapping) Problems() []error {
	problems := make([]error, 0, 3)

	if !m.Has(FieldOriginValue) && !m.Has(FieldOriginCity) {
		problems = append(problems, ErrNoOriginColumn)
	}

	if !m.Has(FieldDestinationValue) && !m.Has(FieldDestinationCity) {
		problems = append(problems, ErrNoDestinationColumn)
	}

	if !m.Has(FieldRate) {
		problems = append(problems, ErrNoRateColumn)
	}

	return problems
}

// Value reads a field out of one row, or empty when the sheet does not carry it
// or the row is short.
func (m Mapping) Value(row []string, field Field) string {
	index, ok := m[field]
	if !ok || index >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[index])
}
