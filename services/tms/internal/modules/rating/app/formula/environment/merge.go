package environment

import (
	goErrors "errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/shopspring/decimal"
)

var ErrNonFiniteValue = goErrors.New("a formula value must be a finite number")

func Merge(
	env map[string]any,
	values map[string]any,
	sources Sources,
	source formulatypes.ValueSource,
) error {
	for key, raw := range values {
		value, err := Exact(raw)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		sources[key] = source

		if strings.Contains(key, ".") {
			formulatypes.SetNestedValue(env, key, value)
			continue
		}

		if _, isDate := env[key].(time.Time); isDate {
			value = coerceDateSample(value)
		}

		if merged, ok := mergeMaps(env[key], value); ok {
			env[key] = merged
			continue
		}

		env[key] = value
	}
	return nil
}

func Exact(value any) (any, error) {
	switch typed := value.(type) {
	case float64:
		exact, ok := decimalutils.FromFloat(typed)
		if !ok {
			return nil, fmt.Errorf("%w: %v", ErrNonFiniteValue, typed)
		}
		return exact, nil
	case float32:
		return Exact(float64(typed))
	case *float64:
		if typed != nil {
			return Exact(*typed)
		}
	case map[string]any:
		exact := make(map[string]any, len(typed))
		for key, item := range typed {
			converted, err := Exact(item)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			exact[key] = converted
		}
		return exact, nil
	case []any:
		exact := make([]any, len(typed))
		for index, item := range typed {
			converted, err := Exact(item)
			if err != nil {
				return nil, fmt.Errorf("[%d]: %w", index, err)
			}
			exact[index] = converted
		}
		return exact, nil
	case int:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case decimal.NullDecimal:
		if typed.Valid {
			return typed.Decimal, nil
		}
	default:
		return value, nil
	}
	return nil, nil //nolint:nilnil // an empty number is the formula's null
}

func mergeMaps(existing, incoming any) (map[string]any, bool) {
	existingMap, ok := existing.(map[string]any)
	if !ok {
		return nil, false
	}
	incomingMap, ok := incoming.(map[string]any)
	if !ok {
		return nil, false
	}

	merged := make(map[string]any, len(existingMap)+len(incomingMap))
	maps.Copy(merged, existingMap)
	for key, value := range incomingMap {
		if nested, isMap := mergeMaps(merged[key], value); isMap {
			merged[key] = nested
			continue
		}
		merged[key] = value
	}

	return merged, true
}

var dateSampleLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
}

func coerceDateSample(value any) any {
	text, ok := value.(string)
	if !ok {
		return value
	}
	for _, layout := range dateSampleLayouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed
		}
	}
	return value
}
