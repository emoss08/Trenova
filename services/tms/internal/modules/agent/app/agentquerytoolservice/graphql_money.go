package agentquerytoolservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

// minorSuffix marks a field that holds an amount in minor units (cents), as
// the schema names every one (netPayMinor). A result shows it as the decimal
// a person reads, under the field's name without the suffix (netPay:
// "1234.56"), and a filter on it takes that decimal, so a model never reads
// or writes cents.
const minorSuffix = "Minor"

// moneyName is the name an amount in minor units is shown under, and whether
// the field is one.
func moneyName(name string) (string, bool) {
	base, ok := strings.CutSuffix(name, minorSuffix)

	return base, ok && base != ""
}

// presentedName is the name a result shows a field under.
func presentedName(name string) string {
	if base, ok := moneyName(name); ok {
		return base
	}

	return name
}

// presentMoney shows each amount in minor units the node carries as its
// decimal, named without the suffix. Where the record already carries the
// amount as a decimal under that name, the minor-unit copy is dropped.
func presentMoney(node, out map[string]any) {
	for key, value := range node {
		base, ok := moneyName(key)
		if !ok {
			continue
		}
		minor, counted := typeutils.IntOf(value)
		if !counted {
			continue
		}
		delete(out, key)
		if _, decimalToo := node[base]; decimalToo {
			continue
		}
		out[base] = minorText(int64(minor))
	}
}

// presentMinor shows the named fields, which hold minor units under a name
// the schema does not mark (a summary's outstandingAdvances), as decimals.
func presentMinor(record map[string]any, fields []string) {
	for _, field := range fields {
		if minor, counted := typeutils.IntOf(record[field]); counted {
			record[field] = minorText(int64(minor))
		}
	}
}

// presentedPath names a field path as a result shows it.
func presentedPath(path string) string {
	parts := strings.Split(path, ".")
	for idx, part := range parts {
		parts[idx] = presentedName(part)
	}

	return strings.Join(parts, ".")
}

// minorUnits reads a decimal a call sent for an amount filter as the minor
// units the field holds.
func minorUnits(raw any, name string) (int64, error) {
	var text string
	switch value := raw.(type) {
	case string:
		text = strings.TrimSpace(value)
	case float64:
		return money.MinorUnits(decimal.NewFromFloat(value)), nil
	case int:
		return money.MinorUnits(decimal.NewFromInt(int64(value))), nil
	default:
		return 0, fmt.Errorf("%s must be an amount such as 1250.00", name)
	}
	amount, err := decimal.NewFromString(text)
	if err != nil {
		return 0, fmt.Errorf("%s must be an amount such as 1250.00", name)
	}

	return money.MinorUnits(amount), nil
}
