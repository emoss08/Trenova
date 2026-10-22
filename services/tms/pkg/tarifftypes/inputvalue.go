package tarifftypes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

var exactJSON = sonic.Config{UseNumber: true, SortMapKeys: true}.Froze()

var ErrNotScalarInput = errors.New("tarifftypes: the input does not take a single number")

type descriptor struct {
	Kind  string      `json:"kind"`
	Unit  string      `json:"unit"`
	Inner *descriptor `json:"inner"`
}

func (d *InputDecl) scalar() (descriptor, bool) {
	var parsed descriptor
	if len(d.Descriptor) > 0 {
		if err := sonic.Unmarshal(d.Descriptor, &parsed); err == nil && parsed.Kind != "" {
			if parsed.Kind == "optional" && parsed.Inner != nil {
				return *parsed.Inner, true
			}
			return parsed, true
		}
	}
	text := strings.TrimSuffix(strings.TrimSpace(d.Type), "?")
	switch {
	case text == "Money":
		return descriptor{Kind: "money"}, true
	case text == "Percent":
		return descriptor{Kind: "percent"}, true
	case text == "Int":
		return descriptor{Kind: "int"}, true
	case text == "Number":
		return descriptor{Kind: "number"}, true
	case strings.HasPrefix(text, "Money/"):
		return descriptor{
			Kind: "quantity",
			Unit: rules.UnitMoney + "/" + strings.TrimPrefix(text, "Money/"),
		}, true
	}
	return descriptor{}, false
}

func (d *InputDecl) ScalarValue(value decimal.Decimal) (rules.RawJSON, error) {
	parsed, ok := d.scalar()
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotScalarInput, d.Name)
	}
	switch parsed.Kind {
	case "money":
		return rules.MoneyValue(value), nil
	case "percent":
		return rules.PercentValue(value), nil
	case "int":
		if !value.IsInteger() {
			return nil, fmt.Errorf("%w: %s takes a whole number", ErrNotScalarInput, d.Name)
		}
		return rules.NumberValue(value), nil
	case "number":
		return rules.NumberValue(value), nil
	case "quantity":
		if parsed.Unit == "" {
			return nil, fmt.Errorf("%w: %s", ErrNotScalarInput, d.Name)
		}
		return rules.QuantityValue(value, parsed.Unit), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrNotScalarInput, d.Name)
	}
}

func FindInput(decls []InputDecl, name string) *InputDecl {
	for i := range decls {
		if decls[i].Name == name {
			return &decls[i]
		}
	}
	return nil
}

func (i Inputs) ScaleMoney(
	factor decimal.Decimal,
	round func(decimal.Decimal) decimal.Decimal,
) (Inputs, error) {
	if i == nil {
		return nil, nil
	}
	out := make(Inputs, len(i))
	for name, raw := range i {
		scaled, err := ScaleMoneyValue(raw, factor, round)
		if err != nil {
			return nil, fmt.Errorf("scale input %s: %w", name, err)
		}
		out[name] = scaled
	}
	return out, nil
}

func ScaleMoneyValue(
	raw rules.RawJSON,
	factor decimal.Decimal,
	round func(decimal.Decimal) decimal.Decimal,
) (rules.RawJSON, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var tree any
	if err := exactJSON.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	scaled, changed, err := scaleTree(tree, factor, round)
	if err != nil {
		return nil, err
	}
	if !changed {
		return raw, nil
	}
	encoded, err := exactJSON.Marshal(scaled)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func scaleTree(
	node any,
	factor decimal.Decimal,
	round func(decimal.Decimal) decimal.Decimal,
) (any, bool, error) {
	switch typed := node.(type) {
	case map[string]any:
		if amount, ok := typed[wireQuantityKey].(string); ok {
			unit, _ := typed["u"].(string)
			if unit != rules.UnitMoney && !strings.HasPrefix(unit, rules.UnitMoney+"/") {
				return typed, false, nil
			}
			value, err := decimal.NewFromString(amount)
			if err != nil {
				return nil, false, err
			}
			return map[string]any{wireQuantityKey: round(value.Mul(factor)).String(), "u": unit}, true, nil
		}
		changed := false
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			scaled, childChanged, err := scaleTree(child, factor, round)
			if err != nil {
				return nil, false, err
			}
			out[key] = scaled
			changed = changed || childChanged
		}
		return out, changed, nil
	case []any:
		changed := false
		out := make([]any, len(typed))
		for index, child := range typed {
			scaled, childChanged, err := scaleTree(child, factor, round)
			if err != nil {
				return nil, false, err
			}
			out[index] = scaled
			changed = changed || childChanged
		}
		return out, changed, nil
	default:
		return node, false, nil
	}
}

func DisplayValue(raw rules.RawJSON) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	if quantity, err := rules.DecodeQuantity(raw); err == nil {
		switch {
		case quantity.Unit == rules.UnitMoney:
			return rules.UnitMoney + quantity.Value.String()
		case quantity.Unit == rules.UnitPercent:
			return quantity.Value.String() + rules.UnitPercent
		case strings.HasPrefix(quantity.Unit, rules.UnitMoney+"/"):
			return rules.UnitMoney + quantity.Value.String() + strings.TrimPrefix(
				quantity.Unit,
				rules.UnitMoney,
			)
		default:
			return quantity.Value.String() + " " + quantity.Unit
		}
	}
	if number, err := rules.DecodeNumber(raw); err == nil {
		return number.String()
	}
	var text string
	if err := sonic.Unmarshal(raw, &text); err == nil {
		return text
	}
	return string(compact(raw))
}

func (i Inputs) Values() (map[string]any, error) {
	out := make(map[string]any, len(i))
	for name, raw := range i {
		var value any
		if err := exactJSON.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("tarifftypes: decode input %s: %w", name, err)
		}
		out[name] = value
	}
	return out, nil
}

func InputsFromValues(values map[string]any) (Inputs, error) {
	if values == nil {
		return nil, nil
	}
	out := make(Inputs, len(values))
	for name, value := range values {
		raw, err := exactJSON.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("tarifftypes: encode input %s: %w", name, err)
		}
		out[name] = raw
	}
	return out, nil
}
