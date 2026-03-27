package tablechangealertservice

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
)

func EvaluateConditions(
	conditions []tablechangealert.Condition,
	conditionMatch string,
	newData map[string]any,
	oldData map[string]any,
	changedFields []string,
) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		result := evaluateCondition(cond, newData, oldData, changedFields)
		if conditionMatch == "any" && result {
			return true
		}
		if conditionMatch == "all" && !result {
			return false
		}
	}

	return conditionMatch == "all"
}

func evaluateCondition(
	cond tablechangealert.Condition,
	newData map[string]any,
	oldData map[string]any,
	changedFields []string,
) bool {
	switch cond.Operator {
	case tablechangealert.OpIsNull:
		v, ok := newData[cond.Field]
		return !ok || v == nil

	case tablechangealert.OpIsNotNull:
		v, ok := newData[cond.Field]
		return ok && v != nil

	case tablechangealert.OpChanged:
		for _, f := range changedFields {
			if f == cond.Field {
				return true
			}
		}
		return false

	case tablechangealert.OpChangedTo:
		newVal, newOk := newData[cond.Field]
		if !newOk || toString(newVal) != toString(cond.Value) {
			return false
		}
		oldVal, oldOk := oldData[cond.Field]
		return !oldOk || toString(oldVal) != toString(cond.Value)

	case tablechangealert.OpChangedFrom:
		oldVal, oldOk := oldData[cond.Field]
		if !oldOk || toString(oldVal) != toString(cond.Value) {
			return false
		}
		newVal, newOk := newData[cond.Field]
		return !newOk || toString(newVal) != toString(cond.Value)

	case tablechangealert.OpEq:
		v, ok := newData[cond.Field]
		if !ok {
			return false
		}
		return toString(v) == toString(cond.Value)

	case tablechangealert.OpNeq:
		v, ok := newData[cond.Field]
		if !ok {
			return true
		}
		return toString(v) != toString(cond.Value)

	case tablechangealert.OpGt, tablechangealert.OpGte, tablechangealert.OpLt, tablechangealert.OpLte:
		v, ok := newData[cond.Field]
		if !ok {
			return false
		}
		fv, fvOk := toFloat64(v)
		cv, cvOk := toFloat64(cond.Value)
		if !fvOk || !cvOk {
			return false
		}
		switch cond.Operator {
		case tablechangealert.OpGt:
			return fv > cv
		case tablechangealert.OpGte:
			return fv >= cv
		case tablechangealert.OpLt:
			return fv < cv
		case tablechangealert.OpLte:
			return fv <= cv
		}

	case tablechangealert.OpContains:
		v, ok := newData[cond.Field]
		if !ok {
			return false
		}
		return strings.Contains(toString(v), toString(cond.Value))

	case tablechangealert.OpNotContains:
		v, ok := newData[cond.Field]
		if !ok {
			return true
		}
		return !strings.Contains(toString(v), toString(cond.Value))
	}

	return false
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toString(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%v", v)
}
