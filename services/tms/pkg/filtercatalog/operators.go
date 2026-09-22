package filtercatalog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

type operatorArity uint8

const (
	arityValue operatorArity = iota
	arityNone
	arityDays
	arityList
)

var operatorArities = map[dbtype.Operator]operatorArity{
	dbtype.OpEqual:              arityValue,
	dbtype.OpNotEqual:           arityValue,
	dbtype.OpGreaterThan:        arityValue,
	dbtype.OpGreaterThanOrEqual: arityValue,
	dbtype.OpLessThan:           arityValue,
	dbtype.OpLessThanOrEqual:    arityValue,
	dbtype.OpContains:           arityValue,
	dbtype.OpStartsWith:         arityValue,
	dbtype.OpEndsWith:           arityValue,
	dbtype.OpIn:                 arityList,
	dbtype.OpNotIn:              arityList,
	dbtype.OpIsNull:             arityNone,
	dbtype.OpIsNotNull:          arityNone,
	dbtype.OpToday:              arityNone,
	dbtype.OpYesterday:          arityNone,
	dbtype.OpTomorrow:           arityNone,
	dbtype.OpLastNDays:          arityDays,
	dbtype.OpNextNDays:          arityDays,
}

var operatorsByKind = map[Kind][]dbtype.Operator{
	KindText: {
		dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpContains, dbtype.OpStartsWith,
		dbtype.OpEndsWith, dbtype.OpIn, dbtype.OpNotIn, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	// isnull and isnotnull belong here for the same reason they belong on text,
	// date and number: an enum column is nullable too, and "has no billing
	// transfer state yet" is one of the most ordinary questions asked of one.
	// Leaving them off sent a caller round the houses building the same answer
	// out of a status filter and a date window.
	KindEnum: {
		dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpIn, dbtype.OpNotIn,
		dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	KindDate: {
		dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual, dbtype.OpLessThan,
		dbtype.OpLessThanOrEqual, dbtype.OpLastNDays, dbtype.OpNextNDays, dbtype.OpToday,
		dbtype.OpYesterday, dbtype.OpTomorrow, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	KindNumber: {
		dbtype.OpEqual, dbtype.OpNotEqual, dbtype.OpGreaterThan, dbtype.OpGreaterThanOrEqual,
		dbtype.OpLessThan, dbtype.OpLessThanOrEqual, dbtype.OpIsNull, dbtype.OpIsNotNull,
	},
	KindBoolean: {dbtype.OpEqual, dbtype.OpNotEqual},
}

func operatorAllowed(kind Kind, operator dbtype.Operator) bool {
	for _, candidate := range operatorsByKind[kind] {
		if candidate == operator {
			return true
		}
	}

	return false
}

func joinOperators(operators []dbtype.Operator) string {
	names := make([]string, 0, len(operators))
	for _, operator := range operators {
		names = append(names, string(operator))
	}

	return strings.Join(names, ", ")
}

// OperatorPhrase renders an operator the way the criteria read back, so an
// empty result says "status equals Active" rather than "status eq".
func OperatorPhrase(operator dbtype.Operator) string {
	switch operator { //nolint:exhaustive // the catalog exposes only these
	case dbtype.OpEqual:
		return "equals"
	case dbtype.OpNotEqual:
		return "is not"
	case dbtype.OpGreaterThan:
		return "after"
	case dbtype.OpGreaterThanOrEqual:
		return "on or after"
	case dbtype.OpLessThan:
		return "before"
	case dbtype.OpLessThanOrEqual:
		return "on or before"
	case dbtype.OpContains:
		return "contains"
	case dbtype.OpStartsWith:
		return "starts with"
	case dbtype.OpEndsWith:
		return "ends with"
	case dbtype.OpIn:
		return "one of"
	case dbtype.OpNotIn:
		return "not one of"
	case dbtype.OpIsNull:
		return "is empty"
	case dbtype.OpIsNotNull:
		return "is set"
	case dbtype.OpToday:
		return "is today"
	case dbtype.OpYesterday:
		return "was yesterday"
	case dbtype.OpTomorrow:
		return "is tomorrow"
	default:
		return string(operator)
	}
}

/*
window turns a relative day count into a bounded range.

OpNextNDays alone applies only `column <= now + N days`, so "expiring in the
next 30 days" would also match every row that lapsed years ago, and OpLastNDays
applies only the floor. Both are fine for a person reading a grid they can sort;
they are not fine for a caller that will summarize the count in a sentence.
*/
func window(
	field Field,
	operator dbtype.Operator,
	days int,
	criteria *Criteria,
) ([]domaintypes.FieldFilter, error) {
	if days <= 0 || days > MaxRelativeDays {
		return nil, fmt.Errorf(
			"%q needs a whole number of days between 1 and %d in the \"days\" field",
			operator, MaxRelativeDays,
		)
	}

	span := int64(days) * SecondsPerDay

	// Windows are whole days in the organization's zone, not offsets from this
	// instant. "Expiring in the next 30 days" starting at the current second
	// excluded a medical card that expired at nine this morning — the one
	// question this was built to answer.
	clk := criteria.Clock
	dayStart := clk.Today()

	lower, upper := dayStart, dayStart+span+SecondsPerDay-1
	phrase := fmt.Sprintf("within the next %d days", days)
	if operator == dbtype.OpLastNDays {
		lower, upper = dayStart-span, clk.Instant()
		phrase = fmt.Sprintf("within the last %d days", days)
	}

	criteria.Field(field.Name, phrase)

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: dbtype.OpGreaterThanOrEqual, Value: lower},
		{Field: field.Name, Operator: dbtype.OpLessThanOrEqual, Value: upper},
	}, nil
}

func listFilter(
	field Field,
	operator dbtype.Operator,
	raw []string,
	criteria *Criteria,
) ([]domaintypes.FieldFilter, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf(
			"%q on %q needs a non-empty \"values\" array", operator, field.Name,
		)
	}

	values := make([]any, 0, len(raw))
	for _, text := range raw {
		value, err := CoerceValue(field, text, criteria.Clock)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}

	criteria.Field(field.Name, OperatorPhrase(operator)+" "+strings.Join(raw, ", "))

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: operator, Value: values},
	}, nil
}

func valueFilter(
	field Field,
	operator dbtype.Operator,
	raw string,
	criteria *Criteria,
) ([]domaintypes.FieldFilter, error) {
	if raw == "" {
		return nil, fmt.Errorf("%q on %q needs a \"value\"", operator, field.Name)
	}

	value, err := CoerceValue(field, raw, criteria.Clock)
	if err != nil {
		return nil, err
	}

	criteria.Field(field.Name, OperatorPhrase(operator)+" "+raw)

	return []domaintypes.FieldFilter{
		{Field: field.Name, Operator: operator, Value: value},
	}, nil
}

// CoerceValue reads one operand in the field's terms.
func CoerceValue(field Field, raw string, clk Clock) (any, error) {
	switch field.Kind {
	case KindEnum:
		for _, candidate := range field.Values {
			if strings.EqualFold(candidate, raw) {
				return candidate, nil
			}
		}

		return nil, fmt.Errorf(
			"%q is not a value of %q; the values are: %s",
			raw, field.Name, strings.Join(field.Values, ", "),
		)
	case KindDate:
		return CoerceDate(field.Name, raw, clk)
	case KindNumber:
		if whole, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return whole, nil
		}
		fractional, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, fmt.Errorf("%q on %q is not a number", raw, field.Name)
		}

		return fractional, nil
	case KindBoolean:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%q on %q must be true or false", raw, field.Name)
		}

		return parsed, nil
	case KindText:
		return raw, nil
	default:
		return raw, nil
	}
}

// CoerceDate reads a calendar date rather than asking the caller for an epoch.
// A fabricated timestamp is indistinguishable from a real one once it is in the
// SQL, and a caller that has to do the arithmetic will sometimes get it wrong
// by a year.
func CoerceDate(name, raw string, clk Clock) (int64, error) {
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return seconds, nil
	}

	if seconds, ok := NamedDay(raw, clk); ok {
		return seconds, nil
	}

	if seconds, ok := clk.ParseDate(raw); ok {
		return seconds, nil
	}

	return 0, fmt.Errorf(
		"%q on %q is not a date; use YYYY-MM-DD, today, tomorrow or yesterday, "+
			"or nextndays/lastndays with a day count",
		raw, name,
	)
}

// NamedDay resolves the words a person uses for a date.
//
// "today" is the obvious thing to send for "expiring from now on", and refusing
// it cost a whole round trip: the caller sent it, read the correction, and
// asked again with the same question. That worked, but only because the refusal
// names what would have worked — on a weaker model the extra turn is where the
// conversation falls apart, and the retry is billed either way.
//
// Resolving it here rather than in a prompt also keeps the clock on the server.
// A model computing today's date is the arithmetic that put a medical card
// three months out of place.
func NamedDay(raw string, clk Clock) (int64, bool) {
	now := clk.Instant()

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "today", "now":
		return clk.DayStart(now), true
	case "tomorrow":
		return clk.DayStart(now + SecondsPerDay), true
	case "yesterday":
		return clk.DayStart(now - SecondsPerDay), true
	default:
		return 0, false
	}
}
