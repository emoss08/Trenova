/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package adapters

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/emoss08/trenova/internal/pkg/sequencegen/consolidationgen"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
)

type ConsolidationFormatProvider struct{}

func NewConsolidationFormatProvider() seqgen.FormatProvider {
	return &ConsolidationFormatProvider{}
}

func (p *ConsolidationFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType sequencestore.SequenceType,
	orgID, buID pulid.ID,
) (*seqgen.Format, error) {
	if sequenceType != sequencestore.SequenceTypeConsolidation {
		return nil, eris.New("invalid sequence type for consolidation provider")
	}

	var seqFormat *sequencegen.SequenceFormat
	var err error

	if buID.IsNil() {
		seqFormat, err = consolidationgen.GetOrganizationConsolidationFormat(ctx, orgID)
		if err != nil {
			seqFormat, err = consolidationgen.GetOrganizationConsolidationFormat(ctx, orgID)
		}
	} else {
		seqFormat, err = consolidationgen.GetConsolidationFormatForBusinessUnit(ctx, orgID, buID)
	}

	if err != nil {
		return nil, eris.Wrap(err, "get consolidation format")
	}

	return &seqgen.Format{
		Type:                    sequencestore.SequenceTypeConsolidation,
		Prefix:                  seqFormat.Prefix,
		IncludeYear:             seqFormat.IncludeYear,
		YearDigits:              seqFormat.YearDigits,
		IncludeMonth:            seqFormat.IncludeMonth,
		SequenceDigits:          seqFormat.SequenceDigits,
		IncludeLocationCode:     seqFormat.IncludeLocationCode,
		LocationCode:            seqFormat.LocationCode,
		IncludeRandomDigits:     seqFormat.IncludeRandomDigits,
		RandomDigitsCount:       seqFormat.RandomDigitsCount,
		IncludeCheckDigit:       seqFormat.IncludeCheckDigit,
		IncludeBusinessUnitCode: seqFormat.IncludeBusinessUnitCode,
		BusinessUnitCode:        seqFormat.BusinessUnitCode,
		UseSeparators:           seqFormat.UseSeparators,
		SeparatorChar:           seqFormat.SeparatorChar,
		AllowCustomFormat:       seqFormat.AllowCustomFormat,
		CustomFormat:            seqFormat.CustomFormat,
	}, nil
}

func ConvertToConsolidationFormat(format *seqgen.Format) *sequencegen.SequenceFormat {
	return &sequencegen.SequenceFormat{
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
