package gqlinput

import (
	"math"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Parser struct {
	errs *errortypes.MultiError
}

func NewParser() *Parser {
	return &Parser{errs: errortypes.NewMultiError()}
}

func (p *Parser) Err() error {
	if p.errs.HasErrors() {
		return p.errs
	}

	return nil
}

func (p *Parser) invalid(field, message string) {
	p.errs.Add(field, errortypes.ErrInvalid, message)
}

func (p *Parser) Decimal(field, raw string) decimal.Decimal {
	if raw == "" {
		p.errs.Add(field, errortypes.ErrRequired, "Value is required")
		return decimal.Zero
	}
	value, err := decimal.NewFromString(raw)
	if err != nil {
		p.invalid(field, "Value must be a number")
		return decimal.Zero
	}

	return value
}

func (p *Parser) NullDecimal(field string, raw *string) decimal.NullDecimal {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(p.Decimal(field, *raw))
}

func (p *Parser) ID(field, raw string) pulid.ID {
	id, err := pulid.MustParse(raw)
	if err != nil {
		p.invalid(field, "Invalid identifier")
		return pulid.Nil
	}

	return id
}

func (p *Parser) OptionalID(field string, raw *string) pulid.ID {
	if raw == nil || *raw == "" {
		return pulid.Nil
	}

	return p.ID(field, *raw)
}

func (p *Parser) IDPtr(field string, raw *string) *pulid.ID {
	if raw == nil || *raw == "" {
		return nil
	}
	id := p.ID(field, *raw)
	if id.IsNil() {
		return nil
	}

	return &id
}

func (p *Parser) Int16(field string, value int) int16 {
	if value < math.MinInt16 || value > math.MaxInt16 {
		p.invalid(field, "Value is out of range")
		return 0
	}

	return int16(value)
}

func (p *Parser) Int32(field string, value int) int32 {
	if value < math.MinInt32 || value > math.MaxInt32 {
		p.invalid(field, "Value is out of range")
		return 0
	}

	return int32(value)
}

func (p *Parser) Int16Ptr(field string, value *int) *int16 {
	if value == nil {
		return nil
	}
	converted := p.Int16(field, *value)

	return &converted
}

func (p *Parser) Int32Ptr(field string, value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := p.Int32(field, *value)

	return &converted
}

func (p *Parser) Int32s(field string, values []int) []int32 {
	converted := make([]int32, 0, len(values))
	for _, value := range values {
		converted = append(converted, p.Int32(field, value))
	}

	return converted
}

func (p *Parser) Number(field string, value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case interface{ Float64() (float64, error) }:
		number, err := v.Float64()
		if err != nil {
			p.invalid(field, "Value must be a number")
		}
		return number
	default:
		p.invalid(field, "Value must be a number")
		return 0
	}
}
