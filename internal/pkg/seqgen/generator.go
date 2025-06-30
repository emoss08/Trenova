package seqgen

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/pkg/logger"
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

	// * Generate the formatted code
	return g.formatCode(format, sequence, now.Year(), int(now.Month())), nil
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

	// * Generate formatted codes
	results := make([]string, 0, len(sequences))
	for _, seq := range sequences {
		code := g.formatCode(format, seq, now.Year(), int(now.Month()))
		results = append(results, code)
	}

	return results, nil
}

// formatCode formats a code based on the format configuration
func (g *generator) formatCode(format *Format, sequence int64, year, month int) string {
	if format.AllowCustomFormat && format.CustomFormat != "" {
		return g.formatCustom(format, sequence, year, month)
	}

	var parts []string

	// * Add prefix
	if format.Prefix != "" {
		parts = append(parts, format.Prefix)
	}

	// * Add business unit code
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		parts = append(parts, format.BusinessUnitCode)
	}

	// * Create date component
	var dateComponent string
	if format.IncludeYear {
		yearStr := strconv.Itoa(year)
		if len(yearStr) > format.YearDigits {
			yearStr = yearStr[len(yearStr)-format.YearDigits:]
		}
		dateComponent += yearStr
	}

	if format.IncludeMonth {
		dateComponent += fmt.Sprintf("%02d", month)
	}

	if dateComponent != "" {
		parts = append(parts, dateComponent)
	}

	// * Add location code
	if format.IncludeLocationCode && format.LocationCode != "" {
		parts = append(parts, format.LocationCode)
	}

	// * Add sequence number
	sequenceFmt := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	parts = append(parts, fmt.Sprintf(sequenceFmt, sequence))

	// * Add random digits
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		parts = append(parts, g.generateRandomDigits(format.RandomDigitsCount))
	}

	// * Add check digit
	if format.IncludeCheckDigit {
		numericPart := strings.Join(parts[1:], "") // Skip prefix for check digit
		checkDigit := g.calculateCheckDigit(numericPart)
		parts = append(parts, strconv.Itoa(checkDigit))
	}

	// * Join with separator
	if format.UseSeparators && format.SeparatorChar != "" {
		return strings.Join(parts, format.SeparatorChar)
	}

	return strings.Join(parts, "")
}

// formatCustom formats using a custom format template
func (g *generator) formatCustom(format *Format, sequence int64, year, month int) string {
	result := format.CustomFormat

	// * Replace placeholders
	replacements := map[string]string{
		"{P}": format.Prefix,
		"{B}": format.BusinessUnitCode,
		"{L}": format.LocationCode,
		"{Y}": g.formatYear(year, format.YearDigits),
		"{M}": fmt.Sprintf("%02d", month),
		"{S}": fmt.Sprintf(fmt.Sprintf("%%0%dd", format.SequenceDigits), sequence),
		"{R}": g.generateRandomDigits(format.RandomDigitsCount),
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// * Calculate check digit if needed
	if strings.Contains(result, "{C}") {
		re := regexp.MustCompile(`\d+`)
		numericParts := re.FindAllString(result, -1)
		numericPart := strings.Join(numericParts, "")
		checkDigit := g.calculateCheckDigit(numericPart)
		result = strings.ReplaceAll(result, "{C}", strconv.Itoa(checkDigit))
	}

	return result
}

// formatYear formats the year according to the specified digits
func (g *generator) formatYear(year, digits int) string {
	yearStr := strconv.Itoa(year)
	if len(yearStr) > digits {
		return yearStr[len(yearStr)-digits:]
	}
	return yearStr
}

// generateRandomDigits generates cryptographically secure random digits
func (g *generator) generateRandomDigits(count int) string {
	if count <= 0 {
		return ""
	}

	maxRandom := 1
	for range count {
		maxRandom *= 10
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxRandom)))
	if err != nil {
		// * Fallback to timestamp-based random
		n = big.NewInt(time.Now().UnixNano() % int64(maxRandom))
	}

	return fmt.Sprintf(fmt.Sprintf("%%0%dd", count), n.Int64())
}

// calculateCheckDigit calculates a Luhn check digit
func (g *generator) calculateCheckDigit(input string) int {
	// * Remove non-digit characters
	re := regexp.MustCompile(`\D`)
	digits := re.ReplaceAllString(input, "")

	sum := 0
	alternate := false

	// * Process from right to left
	for i := len(digits) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(digits[i]))

		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		alternate = !alternate
	}

	return (10 - (sum % 10)) % 10
}
