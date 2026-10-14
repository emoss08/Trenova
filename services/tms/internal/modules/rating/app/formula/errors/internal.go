package errors

import (
	goErrors "errors"
	"fmt"
	"reflect"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string {
	return "an internal error stopped the formula from reading this value"
}

func NewPanicError(value any, stack []byte) *PanicError {
	return &PanicError{Value: value, Stack: stack}
}

func LogFields(err error) []zap.Field {
	fields := make([]zap.Field, 0, 3)

	var resolveErr *ResolveError
	if goErrors.As(err, &resolveErr) && resolveErr.EntityType != "" {
		fields = append(fields, zap.String("entityType", resolveErr.EntityType))
	} else {
		var computeErr *ComputeError
		if goErrors.As(err, &computeErr) && computeErr.EntityType != "" {
			fields = append(fields, zap.String("entityType", computeErr.EntityType))
		}
	}

	var panicErr *PanicError
	if goErrors.As(err, &panicErr) {
		fields = append(fields,
			zap.String("panic", fmt.Sprint(panicErr.Value)),
			zap.ByteString("stack", panicErr.Stack),
		)
	}

	return fields
}

func DescribeValue(value any) string {
	switch value.(type) {
	case nil:
		return "an empty value"
	case bool:
		return "true/false"
	case string:
		return "text"
	case decimal.Decimal, decimal.NullDecimal:
		return "a number"
	case time.Time:
		return "a date"
	case time.Duration:
		return "a duration"
	}

	switch reflect.TypeOf(value).Kind() { //nolint:exhaustive // the default covers the rest
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.String:
		return "text"
	case reflect.Bool:
		return "true/false"
	case reflect.Slice, reflect.Array:
		return "a list"
	case reflect.Map, reflect.Struct:
		return "a group of fields"
	case reflect.Pointer:
		return "a record"
	case reflect.Func:
		return "a function"
	default:
		return "a value of another kind"
	}
}
