package seqgen

import (
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type GeneratorParams struct {
	fx.In

	Store    SequenceStore
	Provider FormatProvider
	Logger   *zap.Logger
}

type generator struct {
	store    SequenceStore
	provider FormatProvider
	l        *zap.Logger
}

func NewGenerator(p GeneratorParams) Generator {
	return &generator{
		store:    p.Store,
		provider: p.Provider,
		l:        p.Logger.Named("seq-generator"),
	}
}

func (g *generator) GenerateShipmentProNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeProNumber,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateConsolidationNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeConsolidation,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateInvoiceNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeInvoice,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateCreditMemoNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeCreditMemo,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateDebitMemoNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeDebitMemo,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateWorkOrderNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeWorkOrder,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateJournalBatchNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeJournalBatch,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateJournalEntryNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeJournalEntry,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) GenerateManualJournalRequestNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
	locationCode, businessUnitCode string,
) (string, error) {
	return g.Generate(ctx, &GenerateRequest{
		Type:             tenant.SequenceTypeManualJournalRequest,
		OrgID:            orgID,
		BuID:             buID,
		LocationCode:     locationCode,
		BusinessUnitCode: businessUnitCode,
	})
}

func (g *generator) Generate(ctx context.Context, req *GenerateRequest) (string, error) {
	if req == nil {
		return "", ErrSequenceRequestRequired
	}

	now := time.Now().UTC()
	if !req.Time.IsZero() {
		now = req.Time.UTC()
	}

	format, err := g.resolveFormat(ctx, req)
	if err != nil {
		return "", err
	}

	sequence, err := g.store.GetNextSequence(ctx, &SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  now.Year(),
		Month: int(now.Month()),
		Count: 1,
	})
	if err != nil {
		return "", fmt.Errorf("get next sequence: %w", err)
	}

	generated, err := g.generateSequenceNumber(format, sequence, now)
	if err != nil {
		return "", err
	}

	if err = g.store.UpdateLastGenerated(ctx, &LastGeneratedRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  now.Year(),
		Month: int(now.Month()),
		Value: generated,
	}); err != nil {
		return "", fmt.Errorf("update last generated: %w", err)
	}

	return generated, nil
}

func (g *generator) GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error) {
	if req == nil {
		return nil, ErrSequenceRequestRequired
	}
	if req.Count <= 0 {
		return []string{}, nil
	}

	now := time.Now().UTC()
	if !req.Time.IsZero() {
		now = req.Time.UTC()
	}

	format, err := g.resolveFormat(ctx, req)
	if err != nil {
		return nil, err
	}

	sequences, err := g.store.GetNextSequenceBatch(ctx, &SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  now.Year(),
		Month: int(now.Month()),
		Count: req.Count,
	})
	if err != nil {
		return nil, fmt.Errorf("get next sequence batch: %w", err)
	}

	results := make([]string, 0, len(sequences))
	for _, sequence := range sequences {
		generated, genErr := g.generateSequenceNumber(format, sequence, now)
		if genErr != nil {
			return nil, genErr
		}
		results = append(results, generated)
	}

	if len(results) > 0 {
		if err = g.store.UpdateLastGenerated(ctx, &LastGeneratedRequest{
			Type:  req.Type,
			OrgID: req.OrgID,
			BuID:  req.BuID,
			Year:  now.Year(),
			Month: int(now.Month()),
			Value: results[len(results)-1],
		}); err != nil {
			return nil, fmt.Errorf("update last generated: %w", err)
		}
	}

	return results, nil
}

func (g *generator) resolveFormat(
	ctx context.Context,
	req *GenerateRequest,
) (*tenant.SequenceFormat, error) {
	var format *tenant.SequenceFormat

	if req.Format != nil {
		format = req.Format
	} else {
		var err error
		format, err = g.provider.GetFormat(ctx, req.Type, req.OrgID, req.BuID)
		if err != nil {
			return nil, fmt.Errorf("get format: %w", err)
		}
	}

	format.LocationCode = req.LocationCode
	format.BusinessUnitCode = req.BusinessUnitCode

	if err := format.Validate(); err != nil {
		return nil, fmt.Errorf("invalid format: %w", err)
	}

	return format, nil
}

func (g *generator) generateSequenceNumber(
	format *tenant.SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	if format == nil {
		return "", ErrSequenceFormatNil
	}

	if format.AllowCustomFormat && format.CustomFormat != "" {
		return generateCustomFormat(format, sequenceNumber, currentTime)
	}

	return generateStandardFormat(format, sequenceNumber, currentTime)
}

func generateStandardFormat(
	format *tenant.SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	components := make([]string, 0, 10)

	if format.Prefix != "" {
		components = append(components, format.Prefix)
	}
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		components = append(components, format.BusinessUnitCode)
	}
	if format.IncludeYear {
		if format.YearDigits == 2 {
			components = append(components, fmt.Sprintf("%02d", currentTime.Year()%100))
		} else {
			components = append(components, fmt.Sprintf("%04d", currentTime.Year()))
		}
	}
	if format.IncludeWeekNumber {
		_, week := currentTime.ISOWeek()
		components = append(components, fmt.Sprintf("%02d", week))
	} else if format.IncludeMonth {
		components = append(components, fmt.Sprintf("%02d", int(currentTime.Month())))
	}
	if format.IncludeDay {
		components = append(components, fmt.Sprintf("%02d", currentTime.Day()))
	}
	if format.IncludeLocationCode && format.LocationCode != "" {
		components = append(components, format.LocationCode)
	}

	seqFmt := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	components = append(components, fmt.Sprintf(seqFmt, sequenceNumber))

	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		rnd, err := randomDigits(format.RandomDigitsCount)
		if err != nil {
			return "", fmt.Errorf("generate random digits: %w", err)
		}
		components = append(components, rnd)
	}

	separator := ""
	if format.UseSeparators {
		separator = format.SeparatorChar
	}

	result := strings.Join(components, separator)
	if format.IncludeCheckDigit {
		digit := luhnCheckDigit(result)
		if separator != "" {
			return result + separator + strconv.Itoa(digit), nil
		}
		return result + strconv.Itoa(digit), nil
	}

	return result, nil
}

func generateCustomFormat(
	format *tenant.SequenceFormat,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	result := format.CustomFormat
	sequencePart := fmt.Sprintf("%0*d", format.SequenceDigits, sequenceNumber)
	year := fmt.Sprintf("%04d", currentTime.Year())
	if format.YearDigits == 2 {
		year = fmt.Sprintf("%02d", currentTime.Year()%100)
	}
	_, week := currentTime.ISOWeek()

	rnd := ""
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		var err error
		rnd, err = randomDigits(format.RandomDigitsCount)
		if err != nil {
			return "", fmt.Errorf("generate random digits: %w", err)
		}
	}

	replacements := map[string]string{
		"{P}": format.Prefix,
		"{B}": format.BusinessUnitCode,
		"{Y}": year,
		"{M}": fmt.Sprintf("%02d", int(currentTime.Month())),
		"{W}": fmt.Sprintf("%02d", week),
		"{D}": fmt.Sprintf("%02d", currentTime.Day()),
		"{L}": format.LocationCode,
		"{S}": sequencePart,
		"{R}": rnd,
	}

	for key, val := range replacements {
		result = strings.ReplaceAll(result, key, val)
	}

	if format.IncludeCheckDigit {
		digit := luhnCheckDigit(result)
		result = strings.ReplaceAll(result, "{C}", strconv.Itoa(digit))
	} else {
		result = strings.ReplaceAll(result, "{C}", "")
	}

	return result, nil
}

func randomDigits(count int) (string, error) {
	if count <= 0 {
		return "", nil
	}

	bytes := make([]byte, count)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	builder := strings.Builder{}
	builder.Grow(count)
	for _, b := range bytes {
		builder.WriteByte('0' + (b % 10))
	}

	return builder.String(), nil
}

func luhnCheckDigit(input string) int {
	sum := 0
	alt := true

	for i := len(input) - 1; i >= 0; i-- {
		ch := input[i]
		if ch < '0' || ch > '9' {
			continue
		}

		num := int(ch - '0')
		if alt {
			num *= 2
			if num > 9 {
				num -= 9
			}
		}
		sum += num
		alt = !alt
	}

	return (10 - (sum % 10)) % 10
}
