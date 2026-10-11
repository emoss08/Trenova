package carrierintel

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

const (
	displayNone            = "none"
	displayYes             = "Yes"
	displayNo              = "No"
	displayNumberDecimals  = 4
	displayPercentDecimals = 2
)

type displayKind uint8

const (
	displayPlain displayKind = iota
	displayDate
	displayMoney
	displayPercent
	displayFraction
)

var displayKindsByPath = map[string]displayKind{
	"insurance.bipdOnFile":    displayMoney,
	"insurance.bipdRequired":  displayMoney,
	"insurance.cargoOnFile":   displayMoney,
	"insurance.cargoRequired": displayMoney,
	"insurance.bondOnFile":    displayMoney,
	"insurance.bondRequired":  displayMoney,
	"safety.riskProbability":  displayFraction,
}

func displayKindFor(path string) displayKind {
	if kind, ok := displayKindsByPath[path]; ok {
		return kind
	}
	field := path[strings.LastIndexByte(path, '.')+1:]
	switch {
	case strings.HasSuffix(field, "At"), strings.HasSuffix(field, "Date"):
		return displayDate
	case strings.HasSuffix(field, "Rate"), strings.HasSuffix(field, "Percent"),
		field == "percentile", field == "threshold":
		return displayPercent
	default:
		return displayPlain
	}
}

func FormatFieldValue(path string, value any) string {
	kind := displayKindFor(path)
	switch typed := value.(type) {
	case nil:
		return displayNone
	case bool:
		if typed {
			return displayYes
		}
		return displayNo
	case string:
		return formatTextValue(kind, typed)
	case decimal.Decimal:
		return formatDecimalValue(kind, typed)
	case *decimal.Decimal:
		if typed == nil {
			return displayNone
		}
		return formatDecimalValue(kind, *typed)
	case float64, float32, int, int64:
		return formatNumberValue(kind, floatutils.FloatValue(typed))
	default:
		return fmt.Sprint(typed)
	}
}

func formatTextValue(kind displayKind, text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return displayNone
	}
	if kind == displayPlain {
		return trimmed
	}
	parsed, err := decimal.NewFromString(trimmed)
	if err != nil {
		return trimmed
	}
	return formatDecimalValue(kind, parsed)
}

func formatDecimalValue(kind displayKind, value decimal.Decimal) string {
	if kind == displayMoney {
		return money.FormatWholeDollars(value)
	}
	return formatNumberValue(kind, value.InexactFloat64())
}

func formatNumberValue(kind displayKind, value float64) string {
	switch kind {
	case displayDate:
		return timeutils.FormatUnixDate(int64(value))
	case displayMoney:
		return money.FormatWholeDollars(decimal.NewFromFloat(value))
	case displayPercent:
		return floatutils.FormatPercent(value, displayPercentDecimals)
	case displayFraction:
		return floatutils.FormatFractionAsPercent(value, displayPercentDecimals)
	default:
		return floatutils.FormatGrouped(value, displayNumberDecimals)
	}
}
