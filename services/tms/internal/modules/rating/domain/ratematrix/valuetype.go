package ratematrix

import (
	"errors"
	"slices"
	"strings"
)

const (
	ValueTypeMoney   = "Money"
	ValueTypePercent = "Percent"
	ValueTypeNumber  = "Number"
	ValueTypeInt     = "Int"

	moneyPerPrefix = "Money/"
)

var ratePerUnits = []string{
	"lb", "cwt", "kg", "t", "ton",
	"mi", "km",
	"in", "ft", "m",
	"ft3", "m3", "gal", "l",
	"min", "h", "d",
	"pallet", "piece", "stop",
}

var errValueType = errors.New(
	"Value type must be Money, Money per a unit (such as Money/cwt or Money/mi), Percent, Number or Int",
)

func IsValueType(text string) bool {
	switch text {
	case ValueTypeMoney, ValueTypePercent, ValueTypeNumber, ValueTypeInt:
		return true
	}
	unit, ok := strings.CutPrefix(text, moneyPerPrefix)
	return ok && slices.Contains(ratePerUnits, unit)
}

func validateValueType(value any) error {
	text, _ := value.(string)
	if IsValueType(text) {
		return nil
	}
	return errValueType
}
