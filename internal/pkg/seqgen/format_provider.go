package seqgen

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

// defaultFormatProvider provides default format configurations
type defaultFormatProvider struct{}

// NewDefaultFormatProvider creates a format provider with default configurations
func NewDefaultFormatProvider() FormatProvider {
	return &defaultFormatProvider{}
}

// GetFormat returns the format configuration for a given sequence type
func (p *defaultFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType sequencestore.SequenceType,
	orgID, buID pulid.ID,
) (*Format, error) {
	// * TODO(wolfred): this should fetch from database
	// * For now, return defaults based on sequence type

	switch sequenceType {
	case sequencestore.SequenceTypeProNumber:
		return &Format{
			Type:                    sequencestore.SequenceTypeProNumber,
			Prefix:                  "S",
			IncludeYear:             true,
			YearDigits:              2,
			IncludeMonth:            true,
			SequenceDigits:          4,
			IncludeLocationCode:     true,
			LocationCode:            "12",
			IncludeRandomDigits:     true,
			RandomDigitsCount:       6,
			IncludeCheckDigit:       false,
			IncludeBusinessUnitCode: false,
			BusinessUnitCode:        "",
			UseSeparators:           false,
			SeparatorChar:           "-",
			AllowCustomFormat:       false,
			CustomFormat:            "{P}{Y}{M}{L}{S}{R}",
		}, nil

	case sequencestore.SequenceTypeConsolidation:
		return &Format{
			Type:                    sequencestore.SequenceTypeConsolidation,
			Prefix:                  "C",
			IncludeYear:             true,
			YearDigits:              2,
			IncludeMonth:            true,
			SequenceDigits:          5,
			IncludeLocationCode:     false,
			LocationCode:            "",
			IncludeRandomDigits:     false,
			RandomDigitsCount:       0,
			IncludeCheckDigit:       true,
			IncludeBusinessUnitCode: true,
			BusinessUnitCode:        "01", // Default, should come from business unit
			UseSeparators:           true,
			SeparatorChar:           "-",
			AllowCustomFormat:       false,
			CustomFormat:            "",
		}, nil

	case sequencestore.SequenceTypeInvoice:
		return &Format{
			Type:                    sequencestore.SequenceTypeInvoice,
			Prefix:                  "INV",
			IncludeYear:             true,
			YearDigits:              4,
			IncludeMonth:            true,
			SequenceDigits:          6,
			IncludeLocationCode:     false,
			LocationCode:            "",
			IncludeRandomDigits:     false,
			RandomDigitsCount:       0,
			IncludeCheckDigit:       false,
			IncludeBusinessUnitCode: false,
			BusinessUnitCode:        "",
			UseSeparators:           true,
			SeparatorChar:           "-",
			AllowCustomFormat:       true,
			CustomFormat:            "{P}-{Y}{M}-{S}",
		}, nil

	case sequencestore.SequenceTypeWorkOrder:
		return &Format{
			Type:                    sequencestore.SequenceTypeWorkOrder,
			Prefix:                  "WO",
			IncludeYear:             true,
			YearDigits:              2,
			IncludeMonth:            false,
			SequenceDigits:          6,
			IncludeLocationCode:     true,
			LocationCode:            "01",
			IncludeRandomDigits:     false,
			RandomDigitsCount:       0,
			IncludeCheckDigit:       false,
			IncludeBusinessUnitCode: false,
			BusinessUnitCode:        "",
			UseSeparators:           false,
			SeparatorChar:           "",
			AllowCustomFormat:       false,
			CustomFormat:            "",
		}, nil

	default:
		return nil, ErrInvalidSequenceType
	}
}
