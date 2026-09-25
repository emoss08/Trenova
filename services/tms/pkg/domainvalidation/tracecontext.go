package domainvalidation

import (
	"errors"

	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	TraceIDLength = 32
	SpanIDLength  = 16
)

func IsTraceID(value string) bool {
	return stringutils.IsLowerHexOfLength(value, TraceIDLength) && !stringutils.IsAllZeros(value)
}

func IsSpanID(value string) bool {
	return stringutils.IsLowerHexOfLength(value, SpanIDLength) && !stringutils.IsAllZeros(value)
}

func TraceID(message string) validation.Rule {
	return stringRule(IsTraceID, message)
}

func SpanID(message string) validation.Rule {
	return stringRule(IsSpanID, message)
}

func stringRule(valid func(string) bool, message string) validation.Rule {
	err := errors.New(message)

	return validation.By(func(value any) error {
		indirect, isNil := validation.Indirect(value)
		if isNil || validation.IsEmpty(indirect) {
			return nil
		}

		text, ok := indirect.(string)
		if !ok || !valid(text) {
			return err
		}

		return nil
	})
}
