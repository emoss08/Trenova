package tarifftypes

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/shopspring/decimal"
)

var ErrNotLiteral = errors.New("tarifftypes: not a Tariff value")

func ParseLiteral(text string) (rules.RawJSON, error) {
	value := strings.TrimSpace(text)
	switch {
	case value == "":
		return nil, fmt.Errorf("%w: the value is empty", ErrNotLiteral)
	case value == "none":
		return rules.NoneValue(), nil
	case strings.HasPrefix(value, "{") || strings.HasPrefix(value, "["):
		if !sonic.ValidString(value) {
			return nil, fmt.Errorf("%w: %q is not valid JSON", ErrNotLiteral, value)
		}
		return rules.RawJSON(value), nil
	case strings.HasSuffix(value, rules.UnitPercent):
		number, err := literalNumber(strings.TrimSuffix(value, rules.UnitPercent), value)
		if err != nil {
			return nil, err
		}
		return rules.PercentValue(number), nil
	}

	negative := strings.HasPrefix(value, "-")
	unsigned := strings.TrimPrefix(value, "-")
	if strings.HasPrefix(unsigned, rules.UnitMoney) {
		amount, per, _ := strings.Cut(strings.TrimPrefix(unsigned, rules.UnitMoney), "/")
		number, err := literalNumber(strings.ReplaceAll(amount, ",", ""), value)
		if err != nil {
			return nil, err
		}
		if negative {
			number = number.Neg()
		}
		if per == "" {
			return rules.MoneyValue(number), nil
		}
		return rules.QuantityValue(number, rules.UnitMoney+"/"+strings.TrimSpace(per)), nil
	}

	if amount, unit, ok := strings.Cut(value, " "); ok {
		number, err := literalNumber(amount, value)
		if err != nil {
			return nil, err
		}
		return rules.QuantityValue(number, strings.TrimSpace(unit)), nil
	}
	number, err := literalNumber(value, value)
	if err != nil {
		return nil, err
	}
	return rules.NumberValue(number), nil
}

func literalNumber(text, whole string) (decimal.Decimal, error) {
	number, err := decimal.NewFromString(strings.TrimSpace(text))
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf(
			"%w: %q is not a number, money ($450, $2.35/mi), percent (12%%), quantity (500 mi) or JSON wire value",
			ErrNotLiteral,
			whole,
		)
	}
	return number, nil
}

func InputsFromLiterals(values map[string]string) (Inputs, error) {
	out := make(Inputs, len(values))
	for name, text := range values {
		if strings.TrimSpace(text) == "" {
			continue
		}
		raw, err := ParseLiteral(text)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		out[name] = raw
	}
	return out, nil
}
