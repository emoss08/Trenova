// Package adapters provides format providers for various sequence types
package adapters

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/sequencestore"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/seqgen"
	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/emoss08/trenova/internal/pkg/sequencegen/consolidationgen"
	"github.com/emoss08/trenova/internal/pkg/sequencegen/pronumbergen"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
)

// * UnifiedFormatProvider provides format configuration for all sequence types
type UnifiedFormatProvider struct {
	logger *logger.Logger
}

// * NewUnifiedFormatProvider creates a new unified format provider
func NewUnifiedFormatProvider(log *logger.Logger) *UnifiedFormatProvider {
	return &UnifiedFormatProvider{
		logger: log,
	}
}

// * GetFormat retrieves the format configuration for a specific sequence type
func (p *UnifiedFormatProvider) GetFormat(
	ctx context.Context,
	sequenceType sequencestore.SequenceType,
	orgID, buID pulid.ID,
) (*seqgen.Format, error) {
	// * Map sequencestore.SequenceType to sequencegen.SequenceType
	genSequenceType, err := mapSequenceType(sequenceType)
	if err != nil {
		return nil, err
	}

	// * Get default format for the sequence type
	defaultFormat, err := sequencegen.GetDefaultFormat(genSequenceType)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get default format")
	}

	// * For specific types, load custom configuration from database
	switch sequenceType {
	case sequencestore.SequenceTypeProNumber:
		format := p.getProNumberFormat(ctx, orgID, buID, defaultFormat)
		return p.convertToSeqgenFormat(format, sequenceType), nil

	case sequencestore.SequenceTypeConsolidation:
		format := p.getConsolidationFormat(ctx, orgID, buID, defaultFormat)
		return p.convertToSeqgenFormat(format, sequenceType), nil

	case sequencestore.SequenceTypeInvoice:
		// * Use default configuration for invoice
		return p.convertToSeqgenFormat(defaultFormat, sequenceType), nil

	case sequencestore.SequenceTypeWorkOrder:
		// * Use default configuration for work order
		return p.convertToSeqgenFormat(defaultFormat, sequenceType), nil

	default:
		// * For unknown types, return error
		return nil, eris.Errorf("unsupported sequence type: %s", sequenceType)
	}
}

// * getProNumberFormat retrieves pro number specific format configuration
func (p *UnifiedFormatProvider) getProNumberFormat(
	ctx context.Context,
	orgID, buID pulid.ID,
	defaultFormat *sequencegen.SequenceFormat,
) *sequencegen.SequenceFormat {
	var format *sequencegen.SequenceFormat
	var err error

	// * Try business unit specific format first
	if !buID.IsNil() {
		format, err = pronumbergen.GetProNumberFormatForBusinessUnit(ctx, orgID, buID)
		if err != nil {
			// * Fall back to organization format
			format, err = pronumbergen.GetOrganizationProNumberFormat(ctx, orgID)
		}
	} else {
		format, err = pronumbergen.GetOrganizationProNumberFormat(ctx, orgID)
	}

	if err != nil {
		p.logger.Debug().
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Msg("pro number format not found, using default")
		return defaultFormat
	}

	return format
}

// * getConsolidationFormat retrieves consolidation specific format configuration
func (p *UnifiedFormatProvider) getConsolidationFormat(
	ctx context.Context,
	orgID, buID pulid.ID,
	defaultFormat *sequencegen.SequenceFormat,
) *sequencegen.SequenceFormat {
	var format *sequencegen.SequenceFormat
	var err error

	if buID.IsNil() {
		format, err = consolidationgen.GetOrganizationConsolidationFormat(ctx, orgID)
	} else {
		format, err = consolidationgen.GetConsolidationFormatForBusinessUnit(ctx, orgID, buID)
	}

	if err != nil {
		p.logger.Debug().
			Str("orgID", orgID.String()).
			Str("buID", buID.String()).
			Msg("consolidation format not found, using default")
		return defaultFormat
	}

	return format
}

// * convertToSeqgenFormat converts sequencegen.SequenceFormat to seqgen.Format
func (p *UnifiedFormatProvider) convertToSeqgenFormat(
	format *sequencegen.SequenceFormat,
	sequenceType sequencestore.SequenceType,
) *seqgen.Format {
	return &seqgen.Format{
		Type:                    sequenceType,
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
		IncludeWeekNumber:       format.IncludeWeekNumber,
		IncludeDay:              format.IncludeDay,
		AllowCustomFormat:       format.AllowCustomFormat,
		CustomFormat:            format.CustomFormat,
	}
}

// * mapSequenceType maps sequencestore.SequenceType to sequencegen.SequenceType
func mapSequenceType(storeType sequencestore.SequenceType) (sequencegen.SequenceType, error) {
	mapping := map[sequencestore.SequenceType]sequencegen.SequenceType{
		sequencestore.SequenceTypeProNumber:     sequencegen.SequenceTypeProNumber,
		sequencestore.SequenceTypeConsolidation: sequencegen.SequenceTypeConsolidation,
		sequencestore.SequenceTypeInvoice:       sequencegen.SequenceTypeInvoice,
		sequencestore.SequenceTypeWorkOrder:     sequencegen.SequenceTypeWorkOrder,
	}

	genType, ok := mapping[storeType]
	if !ok {
		return "", fmt.Errorf("unknown sequence type: %s", storeType)
	}

	return genType, nil
}

// * Ensure UnifiedFormatProvider implements seqgen.FormatProvider
var _ seqgen.FormatProvider = (*UnifiedFormatProvider)(nil)
