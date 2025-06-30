package adapters

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/pkg/pronumbergen"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
)

// ProNumberFormatProvider adapts the existing pronumbergen format functions
// to the seqgen.FormatProvider interface
type ProNumberFormatProvider struct{}

// NewProNumberFormatProvider creates a new pro number format provider
func NewProNumberFormatProvider() seqgen.FormatProvider {
	return &ProNumberFormatProvider{}
}

// GetFormat retrieves the pro number format and converts it to seqgen.Format
func (p *ProNumberFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType sequencestore.SequenceType,
	orgID, buID pulid.ID,
) (*seqgen.Format, error) {
	if sequenceType != sequencestore.SequenceTypeProNumber {
		return nil, eris.New("invalid sequence type for pro number provider")
	}

	var proFormat *pronumbergen.ProNumberFormat
	var err error

	// * Try business unit specific format first
	if !buID.IsNil() {
		proFormat, err = pronumbergen.GetProNumberFormatForBusinessUnit(ctx, orgID, buID)
		if err != nil {
			// * Fall back to organization format
			proFormat, err = pronumbergen.GetOrganizationProNumberFormat(ctx, orgID)
		}
	} else {
		proFormat, err = pronumbergen.GetOrganizationProNumberFormat(ctx, orgID)
	}

	if err != nil {
		return nil, eris.Wrap(err, "get pro number format")
	}

	// * Convert pronumbergen.ProNumberFormat to seqgen.Format
	return &seqgen.Format{
		Type:                    sequencestore.SequenceTypeProNumber,
		Prefix:                  proFormat.Prefix,
		IncludeYear:             proFormat.IncludeYear,
		YearDigits:              proFormat.YearDigits,
		IncludeMonth:            proFormat.IncludeMonth,
		SequenceDigits:          proFormat.SequenceDigits,
		IncludeLocationCode:     proFormat.IncludeLocationCode,
		LocationCode:            proFormat.LocationCode,
		IncludeRandomDigits:     proFormat.IncludeRandomDigits,
		RandomDigitsCount:       proFormat.RandomDigitsCount,
		IncludeCheckDigit:       proFormat.IncludeCheckDigit,
		IncludeBusinessUnitCode: proFormat.IncludeBusinessUnitCode,
		BusinessUnitCode:        proFormat.BusinessUnitCode,
		UseSeparators:           proFormat.UseSeparators,
		SeparatorChar:           proFormat.SeparatorChar,
		AllowCustomFormat:       proFormat.AllowCustomFormat,
		CustomFormat:            proFormat.CustomFormat,
	}, nil
}

// ConvertToProNumberFormat converts a seqgen.Format back to pronumbergen.ProNumberFormat
func ConvertToProNumberFormat(format *seqgen.Format) *pronumbergen.ProNumberFormat {
	return &pronumbergen.ProNumberFormat{
		Prefix:                  format.Prefix,
		IncludeYear:             format.IncludeYear,
		YearDigits:              format.YearDigits,
		IncludeMonth:            format.IncludeMonth,
		SequenceDigits:          format.SequenceDigits,
		IncludeLocationCode:     format.IncludeLocationCode,
		LocationCode:            format.LocationCode,
		IncludeRandomDigits:     format.IncludeRandomDigits,
		RandomDigitsCount:       format.RandomDigitsCount,
		IncludeCheckDigit:       format.IncludeCheckDigit,
		IncludeBusinessUnitCode: format.IncludeBusinessUnitCode,
		BusinessUnitCode:        format.BusinessUnitCode,
		UseSeparators:           format.UseSeparators,
		SeparatorChar:           format.SeparatorChar,
		AllowCustomFormat:       format.AllowCustomFormat,
		CustomFormat:            format.CustomFormat,
	}
}
