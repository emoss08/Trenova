package rules

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/shopspring/decimal"
)

const (
	UnitMoney   = "$"
	UnitPercent = "%"
)

var (
	ErrNotTariffQuantity = errors.New("rules: the value is not a tariff quantity")
	ErrNotTariffNumber   = errors.New("rules: the value is not a tariff number")
)

type TariffQuantity struct {
	Value decimal.Decimal `json:"$q"`
	Unit  string          `json:"u"`
}

func (q TariffQuantity) IsMoney() bool {
	return q.Unit == UnitMoney
}

func (q TariffQuantity) IsMoneyRate() bool {
	return strings.HasPrefix(q.Unit, UnitMoney+"/")
}

func (q TariffQuantity) Raw() RawJSON {
	return QuantityValue(q.Value, q.Unit)
}

type tariffNumber struct {
	Value decimal.Decimal `json:"$n"`
}

type tariffDate struct {
	Value string `json:"$date"`
}

func NumberValue(value decimal.Decimal) RawJSON {
	return encodeTariff(tariffNumber{Value: value})
}

func IntValue(value int64) RawJSON {
	return NumberValue(decimal.NewFromInt(value))
}

func QuantityValue(value decimal.Decimal, unit string) RawJSON {
	return encodeTariff(TariffQuantity{Value: value, Unit: unit})
}

func MoneyValue(value decimal.Decimal) RawJSON {
	return QuantityValue(value, UnitMoney)
}

func PercentValue(value decimal.Decimal) RawJSON {
	return QuantityValue(value, UnitPercent)
}

func DateValue(value time.Time) RawJSON {
	return encodeTariff(tariffDate{Value: value.Format(time.DateOnly)})
}

func DateTimeValue(value time.Time) RawJSON {
	return encodeTariff(tariffDate{Value: value.Format(time.RFC3339)})
}

func TextValue(value string) RawJSON {
	return encodeTariff(value)
}

func BoolValue(value bool) RawJSON {
	if value {
		return RawJSON("true")
	}
	return RawJSON("false")
}

func NoneValue() RawJSON {
	return RawJSON("null")
}

func DecodeQuantity(raw RawJSON) (TariffQuantity, error) {
	var quantity TariffQuantity
	if len(raw) == 0 || string(raw) == "null" {
		return quantity, ErrNotTariffQuantity
	}
	if err := sonic.Unmarshal(raw, &quantity); err != nil {
		return quantity, fmt.Errorf("%w: %w", ErrNotTariffQuantity, err)
	}
	if quantity.Unit == "" {
		return quantity, ErrNotTariffQuantity
	}
	return quantity, nil
}

func DecodeNumber(raw RawJSON) (decimal.Decimal, error) {
	var number struct {
		Value *decimal.Decimal `json:"$n"`
	}
	if err := sonic.Unmarshal(raw, &number); err != nil {
		return decimal.Zero, fmt.Errorf("%w: %w", ErrNotTariffNumber, err)
	}
	if number.Value == nil {
		return decimal.Zero, ErrNotTariffNumber
	}
	return *number.Value, nil
}

func encodeTariff(value any) RawJSON {
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return RawJSON("null")
	}
	return encoded
}

type RawJSON []byte

func (r RawJSON) IsNone() bool {
	return len(r) == 0 || string(r) == "null"
}

func (r RawJSON) MarshalJSON() ([]byte, error) {
	if len(r) == 0 {
		return []byte("null"), nil
	}
	return r, nil
}

func (r *RawJSON) UnmarshalJSON(data []byte) error {
	*r = append((*r)[:0], data...)
	return nil
}
