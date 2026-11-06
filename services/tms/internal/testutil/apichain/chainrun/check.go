package chainrun

import (
	"fmt"
	"reflect"

	"github.com/emoss08/trenova/internal/testutil/apichain"
	"github.com/shopspring/decimal"
)

func sameValue(want, got any) bool {
	if reflect.DeepEqual(want, got) {
		return true
	}
	wantNumber, wantOK := number(want)
	gotNumber, gotOK := number(got)
	if wantOK && gotOK {
		return wantNumber.Equal(gotNumber)
	}
	if isScalar(want) && isScalar(got) {
		return fmt.Sprint(want) == fmt.Sprint(got)
	}

	return false
}

func number(value any) (decimal.Decimal, bool) {
	switch typed := value.(type) {
	case float64:
		return decimal.NewFromFloat(typed), true
	case int:
		return decimal.NewFromInt(int64(typed)), true
	case int64:
		return decimal.NewFromInt(typed), true
	case string:
		parsed, err := decimal.NewFromString(typed)
		return parsed, err == nil
	default:
		return decimal.Decimal{}, false
	}
}

func isScalar(value any) bool {
	switch value.(type) {
	case string, float64, int, int64, bool:
		return true
	default:
		return false
	}
}

func effectRule(effect *apichain.Effect) string {
	switch {
	case effect.Equals != nil:
		return "equals"
	case effect.Increase != nil:
		return "increase"
	case effect.Decrease != nil:
		return "decrease"
	default:
		return "unchanged"
	}
}

func checkEffect(effect *apichain.Effect, expected, before, after any) error {
	switch {
	case effect.Equals != nil:
		if !sameValue(expected, after) {
			return fmt.Errorf("want %v, got %v", expected, after)
		}
	case effect.Unchanged:
		if !sameValue(before, after) {
			return fmt.Errorf("want it unchanged at %v, got %v", before, after)
		}
	default:
		delta, ok := number(expected)
		if !ok {
			return fmt.Errorf("%v is not a number", expected)
		}
		if effect.Decrease != nil {
			delta = delta.Neg()
		}
		start, startOK := number(before)
		end, endOK := number(after)
		if !startOK || !endOK {
			return fmt.Errorf("before %v and after %v must both be numbers", before, after)
		}
		if want := start.Add(delta); !want.Equal(end) {
			return fmt.Errorf("want %s (%s %s %s), got %s",
				want, start, effectRule(effect), delta.Abs(), end)
		}
	}

	return nil
}
