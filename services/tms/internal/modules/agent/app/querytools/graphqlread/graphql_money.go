package graphqlread

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/typeutils"
	"github.com/shopspring/decimal"
)

// minorSuffix marks a field that holds an amount in minor units (cents), as
// the schema names every one (netPayMinor). A spec that sets amounts shows
// each as the decimal a person reads, under the field's name without the
// suffix (netPay: "1234.56"), and filters it by that decimal, so a model
// never reads or writes cents.
const minorSuffix = "Minor"

// MoneyName is the name an amount in minor units is shown under, and whether
// the field is one.
func MoneyName(name string) (string, bool) {
	base, ok := strings.CutSuffix(name, minorSuffix)

	return base, ok && base != ""
}

// PresentedName is the name a result that shows amounts shows a field under.
func PresentedName(name string) string {
	if base, ok := MoneyName(name); ok {
		return base
	}

	return name
}

// PresentAmounts shows each amount in minor units the record and the records
// it carries hold as its decimal, named without the suffix. Where a record
// already carries the amount as a decimal under that name, the minor-unit
// copy is dropped.
func PresentAmounts(record map[string]any) {
	for key, value := range record {
		switch v := value.(type) {
		case map[string]any:
			PresentAmounts(v)
		case []any:
			for _, element := range v {
				if object, ok := element.(map[string]any); ok {
					PresentAmounts(object)
				}
			}
		}
		base, ok := MoneyName(key)
		if !ok {
			continue
		}
		minor, counted := typeutils.IntOf(value)
		if !counted {
			continue
		}
		delete(record, key)
		if _, decimalToo := record[base]; decimalToo {
			continue
		}
		record[base] = querytool.MinorText(int64(minor))
	}
}

// PresentMinor shows the named fields, which hold minor units under a name
// the schema does not mark (a summary's outstandingAdvances), as decimals.
func PresentMinor(record map[string]any, fields []string) {
	for _, field := range fields {
		if minor, counted := typeutils.IntOf(record[field]); counted {
			record[field] = querytool.MinorText(int64(minor))
		}
	}
}

// PresentedPaths name withheld field paths as a result that shows amounts
// names them (lines.amountMinor is lines.amount), once each.
func PresentedPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		parts := strings.Split(path, ".")
		for idx, part := range parts {
			parts[idx] = PresentedName(part)
		}
		named := strings.Join(parts, ".")
		if !slices.Contains(out, named) {
			out = append(out, named)
		}
	}

	return out
}

// AmountParams turns the range parameters a spec that shows amounts derived
// from fields held in minor units into the decimal a result shows: amountMin
// and amountMax on amountMinor, compiled back to minor units.
func AmountParams(params []Param) []Param {
	out := slices.Clone(params)
	for idx := range out {
		param := &out[idx]
		if param.OnInput || (param.Kind != ParamAtLeast && param.Kind != ParamAtMost) {
			continue
		}
		base, ok := MoneyName(param.Field)
		if !ok || strings.Contains(param.Field, ".") {
			continue
		}
		suffix := suffixMin
		bound := "at least"
		if param.Kind == ParamAtMost {
			suffix, bound = SuffixMax, "at most"
		}
		words := textcase.HumanizeSnakeCase(textcase.ConvertCamelToSnake(base))
		param.Name = base + suffix
		param.Minor = true
		param.Schema = map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "Only records whose " + words + " is " + bound +
				" this. Write it as a decimal in the record's currency, such as 1250.00.",
		}
	}

	return out
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
