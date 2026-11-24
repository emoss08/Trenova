package boardcatalog

import (
	"slices"
	"strings"

	"github.com/shopspring/decimal"
)

type ValueKind string

const (
	ValueCount    = ValueKind("count")
	ValueMoney    = ValueKind("money")
	ValuePercent  = ValueKind("percent")
	ValueDuration = ValueKind("duration")
	ValueDays     = ValueKind("days")
	ValueHours    = ValueKind("hours")
	ValueNumber   = ValueKind("number")
	ValueText     = ValueKind("text")
)

type Value struct {
	Kind     ValueKind       `json:"kind"`
	Number   decimal.Decimal `json:"number"`
	Currency string          `json:"currency,omitempty"`
	Text     string          `json:"text,omitempty"`
}

func (v *Value) Same(other *Value) bool {
	if v == nil || other == nil {
		return v == other
	}

	return v.Kind == other.Kind && v.Currency == other.Currency && v.Text == other.Text &&
		v.Number.Equal(other.Number)
}

type Tone string

const (
	ToneBad  = Tone("bad")
	ToneWarn = Tone("warn")
	ToneInfo = Tone("info")
	ToneOK   = Tone("ok")
)

func AllTones() []Tone {
	return []Tone{ToneBad, ToneWarn, ToneInfo, ToneOK}
}

func (t Tone) IsValid() bool {
	return slices.Contains(AllTones(), t)
}

func (t Tone) Severity() int {
	switch t {
	case ToneBad:
		return 3
	case ToneWarn:
		return 2
	case ToneInfo:
		return 1
	case ToneOK:
	}

	return 0
}

const minorExponent = -2

func Money(minor int64, currency string) Value {
	return Value{
		Kind:     ValueMoney,
		Number:   decimal.New(minor, minorExponent),
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

func Amount(amount decimal.Decimal, currency string) Value {
	return Value{
		Kind:     ValueMoney,
		Number:   amount,
		Currency: strings.ToUpper(strings.TrimSpace(currency)),
	}
}

func Duration(seconds int64) Value {
	return Value{Kind: ValueDuration, Number: decimal.NewFromInt(seconds)}
}

func Hours(seconds int64) Value {
	return Value{Kind: ValueHours, Number: decimal.NewFromInt(seconds)}
}

func Days(days int64) Value {
	return Value{Kind: ValueDays, Number: decimal.NewFromInt(days)}
}

func Count(n int64) Value {
	return Value{Kind: ValueCount, Number: decimal.NewFromInt(n)}
}

func Percent(fraction decimal.Decimal) Value {
	return Value{Kind: ValuePercent, Number: fraction}
}

func Number(number decimal.Decimal) Value {
	return Value{Kind: ValueNumber, Number: number}
}
