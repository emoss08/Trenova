package seqgen

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FormatProviderParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type formatProvider struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewFormatProvider(p FormatProviderParams) FormatProvider {
	return &formatProvider{
		db: p.DB,
		l:  p.Logger.Named("seq-format-provider"),
	}
}

func (p *formatProvider) GetFormat(
	ctx context.Context,
	sequenceType tenant.SequenceType,
	orgID, buID pulid.ID,
) (*tenant.SequenceFormat, error) {
	cfg := new(tenant.SequenceConfig)
	err := p.db.DB().NewSelect().
		Model(cfg).
		Where("sequence_type = ?", sequenceType).
		Where("organization_id = ?", orgID).
		Where("business_unit_id = ?", buID).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return tenant.DefaultSequenceFormat(sequenceType)
		}
		return nil, fmt.Errorf("find sequence config: %w", err)
	}

	format := &tenant.SequenceFormat{
		Type:                    cfg.SequenceType,
		Prefix:                  cfg.Prefix,
		IncludeYear:             cfg.IncludeYear,
		YearDigits:              int(cfg.YearDigits),
		IncludeMonth:            cfg.IncludeMonth,
		IncludeWeekNumber:       cfg.IncludeWeekNumber,
		IncludeDay:              cfg.IncludeDay,
		SequenceDigits:          int(cfg.SequenceDigits),
		IncludeLocationCode:     cfg.IncludeLocationCode,
		IncludeRandomDigits:     cfg.IncludeRandomDigits,
		RandomDigitsCount:       int(cfg.RandomDigitsCount),
		IncludeCheckDigit:       cfg.IncludeCheckDigit,
		IncludeBusinessUnitCode: cfg.IncludeBusinessUnitCode,
		UseSeparators:           cfg.UseSeparators,
		SeparatorChar:           cfg.SeparatorChar,
		AllowCustomFormat:       cfg.AllowCustomFormat,
		CustomFormat:            cfg.CustomFormat,
	}

	return format, nil
}
