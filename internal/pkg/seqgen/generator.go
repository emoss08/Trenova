package seqgen

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type GeneratorParams struct {
	fx.In

	Store    SequenceStore
	Provider FormatProvider
	Logger   *logger.Logger
}

// generator implements the Generator interface
type generator struct {
	store    SequenceStore
	provider FormatProvider
	l        *zerolog.Logger
}

// NewGenerator creates a new code generator
func NewGenerator(p GeneratorParams) Generator {
	l := p.Logger.With().
		Str("component", "seqGenerator").
		Logger()

	return &generator{
		store:    p.Store,
		provider: p.Provider,
		l:        &l,
	}
}

// Generate creates a single formatted code
func (g *generator) Generate(ctx context.Context, req *GenerateRequest) (string, error) {
	// * Get the format configuration
	format, err := g.provider.GetFormat(ctx, req.Type, req.OrganizationID, req.BusinessUnitID)
	if err != nil {
		return "", eris.Wrap(err, "get format configuration")
	}

	// * Get the next sequence number
	now := time.Now()
	seqReq := &SequenceRequest{
		Type:           req.Type,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		Year:           now.Year(),
		Month:          int(now.Month()),
		Count:          1,
	}

	sequence, err := g.store.GetNextSequence(ctx, seqReq)
	if err != nil {
		return "", eris.Wrap(err, "get next sequence")
	}

	// * Convert seqgen.Format to sequencegen.SequenceFormat
	seqFormat := convertToSequenceFormat(format)

	// * Generate the formatted code using unified generator
	code, err := sequencegen.GenerateSequenceNumber(ctx, seqFormat, sequence, now)
	if err != nil {
		return "", eris.Wrap(err, "generate sequence number")
	}

	return code, nil
}

// GenerateBatch creates multiple formatted codes
func (g *generator) GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error) {
	if req.Count <= 0 {
		return []string{}, nil
	}

	// * Get the format configuration
	format, err := g.provider.GetFormat(ctx, req.Type, req.OrganizationID, req.BusinessUnitID)
	if err != nil {
		return nil, eris.Wrap(err, "get format configuration")
	}

	// * Get batch of sequence numbers
	now := time.Now()
	seqReq := &SequenceRequest{
		Type:           req.Type,
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		Year:           now.Year(),
		Month:          int(now.Month()),
		Count:          req.Count,
	}

	sequences, err := g.store.GetNextSequenceBatch(ctx, seqReq)
	if err != nil {
		return nil, eris.Wrap(err, "get sequence batch")
	}

	// * Convert seqgen.Format to sequencegen.SequenceFormat
	seqFormat := convertToSequenceFormat(format)

	// * Generate formatted codes
	results := make([]string, 0, len(sequences))
	for _, seq := range sequences {
		code, genErr := sequencegen.GenerateSequenceNumber(ctx, seqFormat, seq, now)
		if genErr != nil {
			return nil, eris.Wrap(genErr, "generate sequence number")
		}
		results = append(results, code)
	}

	return results, nil
}

// convertToSequenceFormat converts seqgen.Format to sequencegen.SequenceFormat
func convertToSequenceFormat(format *Format) *sequencegen.SequenceFormat {
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
		IncludeWeekNumber:       format.IncludeWeekNumber,
		IncludeDay:              format.IncludeDay,
		AllowCustomFormat:       format.AllowCustomFormat,
		CustomFormat:            format.CustomFormat,
	}
}
