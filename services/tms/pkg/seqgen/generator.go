package seqgen

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
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
	store       SequenceStore
	provider    FormatProvider
	l           *zap.Logger
	cacheMu     sync.RWMutex
	formatCache map[string]*formatCacheEntry
	cacheTTL    time.Duration
}

type formatCacheEntry struct {
	format    *Format
	expiresAt time.Time
}

func formatCacheKey(t SequenceType, orgID, buID string) string {
	return fmt.Sprintf("%s:%s:%s", t, orgID, buID)
}

func NewGenerator(p GeneratorParams) Generator {
	return &generator{
		store:       p.Store,
		provider:    p.Provider,
		l:           p.Logger.With(zap.String("component", "seqGenerator")),
		formatCache: make(map[string]*formatCacheEntry),
		cacheTTL:    15 * time.Minute, // Default cache TTL
	}
}

func (g *generator) GenerateShipmentProNumber(
	ctx context.Context,
	orgID, buID pulid.ID,
) (string, error) {
	format := DefaultShipmentProNumberFormat()
	req := &GenerateRequest{
		Type:   SequenceTypeProNumber,
		OrgID:  orgID,
		BuID:   buID,
		Format: format,
	}
	return g.Generate(ctx, req)
}

func (g *generator) GenerateShipmentProNumberBatch(
	ctx context.Context,
	orgID, buID pulid.ID,
	count int,
) ([]string, error) {
	format := DefaultShipmentProNumberFormat()
	req := &GenerateRequest{
		Type:   SequenceTypeProNumber,
		OrgID:  orgID,
		BuID:   buID,
		Count:  count,
		Format: format,
	}
	return g.GenerateBatch(ctx, req)
}

func (g *generator) Generate(ctx context.Context, req *GenerateRequest) (string, error) {
	var format *Format
	if req.Format != nil {
		format = req.Format
	} else {
		var err error
		format, err = g.getCachedFormat(ctx, req.Type, req.OrgID, req.BuID)
		if err != nil {
			return "", fmt.Errorf("get format configuration: %w", err)
		}
	}

	now := time.Now()
	if !req.Time.IsZero() {
		now = req.Time
	}

	seqReq := &SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  now.Year(),
		Month: int(now.Month()),
		Count: 1,
	}

	sequence, err := g.store.GetNextSequence(ctx, seqReq)
	if err != nil {
		return "", fmt.Errorf("get next sequence: %w", err)
	}

	code, err := g.generateSequenceNumber(format, sequence, now)
	if err != nil {
		return "", fmt.Errorf("generate sequence number: %w", err)
	}

	if err = g.ValidateSequence(code, format); err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	return code, nil
}

func (g *generator) GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error) {
	if req.Count <= 0 {
		return []string{}, nil
	}

	var format *Format
	if req.Format != nil {
		format = req.Format
	} else {
		var err error
		format, err = g.getCachedFormat(ctx, req.Type, req.OrgID, req.BuID)
		if err != nil {
			return nil, fmt.Errorf("get format configuration: %w", err)
		}
	}

	now := time.Now()
	seqReq := &SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  now.Year(),
		Month: int(now.Month()),
		Count: req.Count,
	}

	sequences, err := g.store.GetNextSequenceBatch(ctx, seqReq)
	if err != nil {
		return nil, fmt.Errorf("get sequence batch: %w", err)
	}

	results := make([]string, 0, len(sequences))
	for _, seq := range sequences {
		code, genErr := g.generateSequenceNumber(format, seq, now)
		if genErr != nil {
			return nil, fmt.Errorf("generate sequence number: %w", genErr)
		}
		results = append(results, code)
	}

	return results, nil
}

func (g *generator) generateSequenceNumber(
	format *Format,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	if format == nil {
		return "", ErrSequenceFormatNil
	}

	if err := format.Validate(); err != nil {
		return "", fmt.Errorf("invalid sequence format: %w", err)
	}

	if format.AllowCustomFormat && format.CustomFormat != "" {
		return g.generateCustomFormat(format, sequenceNumber, currentTime)
	}

	return g.generateStandardFormat(format, sequenceNumber, currentTime)
}

func (g *generator) generateStandardFormat(
	format *Format,
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
		year := utils.GetYearString(currentTime, format.YearDigits)
		components = append(components, year)
	}

	if format.IncludeWeekNumber {
		_, week := currentTime.ISOWeek()
		components = append(components, fmt.Sprintf("%02d", week))
	} else if format.IncludeMonth {
		components = append(components, fmt.Sprintf("%02d", currentTime.Month()))
	}

	if format.IncludeDay {
		components = append(components, fmt.Sprintf("%02d", currentTime.Day()))
	}

	if format.IncludeLocationCode && format.LocationCode != "" {
		components = append(components, format.LocationCode)
	}

	seqFormat := fmt.Sprintf("%%0%dd", format.SequenceDigits)
	components = append(components, fmt.Sprintf(seqFormat, sequenceNumber))

	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		randomDigits := utils.GenerateRandomDigits(format.RandomDigitsCount)
		components = append(components, randomDigits)
	}

	separator := ""
	if format.UseSeparators && format.SeparatorChar != "" {
		separator = format.SeparatorChar
	}
	result := strings.Join(components, separator)

	if format.IncludeCheckDigit {
		checkDigit := utils.CalculateLuhnCheckDigit(result)
		if separator != "" {
			return result + separator + strconv.Itoa(checkDigit), nil
		}
		return result + strconv.Itoa(checkDigit), nil
	}

	return result, nil
}

func (g *generator) generateCustomFormat(
	format *Format,
	sequenceNumber int64,
	currentTime time.Time,
) (string, error) {
	result := format.CustomFormat

	replacements := map[string]string{
		"{P}": format.Prefix,
		"{Y}": utils.GetYearString(currentTime, format.YearDigits),
		"{M}": fmt.Sprintf("%02d", currentTime.Month()),
		"{W}": fmt.Sprintf("%02d", utils.GetISOWeek(currentTime)),
		"{D}": fmt.Sprintf("%02d", currentTime.Day()),
		"{L}": format.LocationCode,
		"{B}": format.BusinessUnitCode,
		"{S}": fmt.Sprintf(fmt.Sprintf("%%0%dd", format.SequenceDigits), sequenceNumber),
		"{R}": utils.GenerateRandomDigits(format.RandomDigitsCount),
	}

	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	if strings.Contains(result, "{C}") {
		tempResult := strings.ReplaceAll(result, "{C}", "")
		checkDigit := utils.CalculateLuhnCheckDigit(tempResult)
		result = strings.ReplaceAll(result, "{C}", strconv.Itoa(checkDigit))
	}

	return result, nil
}

func (g *generator) getCachedFormat(
	ctx context.Context,
	seqType SequenceType,
	orgID, buID pulid.ID,
) (*Format, error) {
	key := formatCacheKey(seqType, orgID.String(), buID.String())

	g.cacheMu.RLock()
	entry, found := g.formatCache[key]
	g.cacheMu.RUnlock()

	if found && entry.expiresAt.After(time.Now()) {
		g.l.Debug("format cache hit", zap.String("key", key))
		return entry.format, nil
	}

	g.l.Debug("format cache miss", zap.String("key", key))
	format, err := g.provider.GetFormat(ctx, seqType, orgID, buID)
	if err != nil {
		return nil, err
	}

	g.cacheMu.Lock()
	g.formatCache[key] = &formatCacheEntry{
		format:    format,
		expiresAt: time.Now().Add(g.cacheTTL),
	}
	g.cacheMu.Unlock()

	return format, nil
}

func (g *generator) ClearCache() {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()
	g.formatCache = make(map[string]*formatCacheEntry)
}

func (g *generator) SetCacheTTL(ttl time.Duration) {
	g.cacheTTL = ttl
}

func (g *generator) ValidateSequence(sequence string, format *Format) error {
	if err := g.validateBasicRequirements(sequence, format); err != nil {
		return err
	}
	if err := g.validatePrefixAndSeparators(sequence, format); err != nil {
		return err
	}
	if err := g.validateSequenceLength(sequence, format); err != nil {
		return err
	}
	if format.IncludeCheckDigit {
		return g.validateCheckDigit(sequence, format)
	}
	return nil
}

func (g *generator) validateBasicRequirements(sequence string, format *Format) error {
	if sequence == "" {
		return ErrSequenceCannotBeEmpty
	}
	if format == nil {
		return ErrSequenceFormatNil
	}
	return nil
}

func (g *generator) validatePrefixAndSeparators(sequence string, format *Format) error {
	if err := g.validatePrefix(sequence, format); err != nil {
		return err
	}
	return g.validateSeparators(sequence, format)
}

func (g *generator) validatePrefix(sequence string, format *Format) error {
	if format.Prefix != "" && !strings.HasPrefix(sequence, format.Prefix) {
		return fmt.Errorf(
			"sequence should start with prefix %q but got %q",
			format.Prefix,
			sequence,
		)
	}
	return nil
}

func (g *generator) validateSeparators(sequence string, format *Format) error {
	if format.UseSeparators {
		return nil
	}
	allSeparators := make([]string, len(AllowedSeparators), len(AllowedSeparators)+2)
	copy(allSeparators, AllowedSeparators)
	allSeparators = append(allSeparators, " ", "|")
	for _, sep := range allSeparators {
		if strings.Contains(sequence, sep) {
			return fmt.Errorf(
				"sequence contains unexpected separator %q but format specifies no separators",
				sep,
			)
		}
	}
	return nil
}

func (g *generator) validateSequenceLength(sequence string, format *Format) error {
	minLength := g.calculateMinimumLength(format)
	if len(sequence) < minLength {
		return fmt.Errorf(
			"sequence length %d is less than expected minimum %d",
			len(sequence),
			minLength,
		)
	}
	return nil
}

func (g *generator) calculateMinimumLength(format *Format) int {
	minLength := len(format.Prefix)

	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		minLength += len(format.BusinessUnitCode)
	}
	if format.IncludeYear {
		minLength += format.YearDigits
	}
	if format.IncludeMonth || format.IncludeWeekNumber {
		minLength += 2
	}
	if format.IncludeDay {
		minLength += 2
	}
	if format.IncludeLocationCode && format.LocationCode != "" {
		minLength += len(format.LocationCode)
	}

	minLength += format.SequenceDigits

	if format.IncludeRandomDigits {
		minLength += format.RandomDigitsCount
	}
	if format.IncludeCheckDigit {
		minLength++
	}
	if format.UseSeparators && format.SeparatorChar != "" {
		separatorLength := g.calculateSeparatorLength(format)
		minLength += separatorLength
	}

	return minLength
}

func (g *generator) calculateSeparatorLength(format *Format) int {
	componentCount := g.countSequenceComponents(format)
	if componentCount > 1 {
		return (componentCount - 1) * len(format.SeparatorChar)
	}
	return 0
}

func (g *generator) countSequenceComponents(format *Format) int {
	componentCount := 0
	if format.Prefix != "" {
		componentCount++
	}
	if format.IncludeBusinessUnitCode && format.BusinessUnitCode != "" {
		componentCount++
	}
	if format.IncludeYear || format.IncludeMonth || format.IncludeWeekNumber || format.IncludeDay {
		componentCount++
	}
	if format.IncludeLocationCode && format.LocationCode != "" {
		componentCount++
	}
	componentCount++ // Always include sequence number
	if format.IncludeRandomDigits {
		componentCount++
	}
	if format.IncludeCheckDigit {
		componentCount++
	}
	return componentCount
}

func (g *generator) validateCheckDigit(sequence string, format *Format) error {
	baseSequence, err := g.extractBaseSequence(sequence, format)
	if err != nil {
		return err
	}

	expectedCheckDigit := utils.CalculateLuhnCheckDigit(baseSequence)
	actualCheckDigit := sequence[len(sequence)-1:]

	if actualCheckDigit != strconv.Itoa(expectedCheckDigit) {
		return fmt.Errorf(
			"invalid check digit: expected %d, got %s",
			expectedCheckDigit,
			actualCheckDigit,
		)
	}
	return nil
}

func (g *generator) extractBaseSequence(sequence string, format *Format) (string, error) {
	if format.UseSeparators && format.SeparatorChar != "" {
		return g.extractBaseSequenceWithSeparators(sequence, format)
	}
	return g.extractBaseSequenceWithoutSeparators(sequence)
}

func (g *generator) extractBaseSequenceWithSeparators(
	sequence string,
	format *Format,
) (string, error) {
	parts := strings.Split(sequence, format.SeparatorChar)
	if len(parts) > 1 {
		checkDigitStr := parts[len(parts)-1]
		if len(checkDigitStr) != 1 {
			return "", ErrInvalidCheckDigitFormat
		}
		return strings.Join(parts[:len(parts)-1], format.SeparatorChar), nil
	}
	return g.extractBaseSequenceWithoutSeparators(sequence)
}

func (g *generator) extractBaseSequenceWithoutSeparators(sequence string) (string, error) {
	if len(sequence) < 2 {
		return "", ErrSequenceTooShort
	}
	return sequence[:len(sequence)-1], nil
}

func (g *generator) ParseSequence(sequence string, format *Format) (*SequenceComponents, error) {
	if format == nil {
		return nil, ErrSequenceFormatNil
	}

	if format.AllowCustomFormat && format.CustomFormat != "" {
		return g.parseCustomFormat(sequence, format)
	}

	return g.parseStandardFormat(sequence, format)
}

func (g *generator) parseStandardFormat(
	sequence string,
	format *Format,
) (*SequenceComponents, error) {
	components := &SequenceComponents{
		Original: sequence,
	}

	parser := &sequenceParser{
		sequence:  sequence,
		pos:       0,
		separator: g.getSeparator(format),
	}

	return g.extractAllComponents(parser, format, components)
}

func (g *generator) getSeparator(format *Format) string {
	if format.UseSeparators && format.SeparatorChar != "" {
		return format.SeparatorChar
	}
	return ""
}

func (g *generator) extractAllComponents(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) (*SequenceComponents, error) {
	if err := g.extractPrefix(parser, format, components); err != nil {
		return nil, err
	}
	if err := g.extractBusinessUnit(parser, format, components); err != nil {
		return nil, err
	}
	if err := g.extractDateComponents(parser, format, components); err != nil {
		return nil, err
	}
	if err := g.extractLocationAndSequence(parser, format, components); err != nil {
		return nil, err
	}
	return g.extractOptionalComponents(parser, format, components)
}

func (g *generator) extractPrefix(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) error {
	if format.Prefix == "" {
		return nil
	}
	prefix, err := parser.extractNext(len(format.Prefix))
	if err != nil {
		return fmt.Errorf("extract prefix: %w", err)
	}
	if prefix != format.Prefix {
		return fmt.Errorf("prefix mismatch: expected %q, got %q", format.Prefix, prefix)
	}
	components.Prefix = prefix
	return nil
}

func (g *generator) extractBusinessUnit(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) error {
	if !format.IncludeBusinessUnitCode || format.BusinessUnitCode == "" {
		return nil
	}
	buCode, err := parser.extractNext(len(format.BusinessUnitCode))
	if err != nil {
		return fmt.Errorf("extract business unit code: %w", err)
	}
	components.BusinessUnitCode = buCode
	return nil
}

func (g *generator) extractDateComponents(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) error {
	if format.IncludeYear {
		year, err := parser.extractNext(format.YearDigits)
		if err != nil {
			return fmt.Errorf("extract year: %w", err)
		}
		components.Year = year
	}

	if format.IncludeWeekNumber {
		week, err := parser.extractNext(2)
		if err != nil {
			return fmt.Errorf("extract week: %w", err)
		}
		components.Week = week
	} else if format.IncludeMonth {
		month, err := parser.extractNext(2)
		if err != nil {
			return fmt.Errorf("extract month: %w", err)
		}
		components.Month = month
	}

	if format.IncludeDay {
		day, err := parser.extractNext(2)
		if err != nil {
			return fmt.Errorf("extract day: %w", err)
		}
		components.Day = day
	}
	return nil
}

func (g *generator) extractLocationAndSequence(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) error {
	if format.IncludeLocationCode && format.LocationCode != "" {
		locationCode, err := parser.extractNext(len(format.LocationCode))
		if err != nil {
			return fmt.Errorf("extract location code: %w", err)
		}
		components.LocationCode = locationCode
	}

	seq, err := parser.extractNext(format.SequenceDigits)
	if err != nil {
		return fmt.Errorf("extract sequence: %w", err)
	}
	components.Sequence = seq
	return nil
}

func (g *generator) extractOptionalComponents(
	parser *sequenceParser,
	format *Format,
	components *SequenceComponents,
) (*SequenceComponents, error) {
	if format.IncludeRandomDigits && format.RandomDigitsCount > 0 {
		random, err := parser.extractNext(format.RandomDigitsCount)
		if err != nil {
			return nil, fmt.Errorf("extract random digits: %w", err)
		}
		components.RandomDigits = random
	}

	if format.IncludeCheckDigit {
		if parser.pos >= len(parser.sequence) {
			return nil, ErrMissingCheckDigit
		}
		components.CheckDigit = parser.sequence[parser.pos:]
	}

	return components, nil
}

type sequenceParser struct {
	sequence  string
	pos       int
	separator string
}

func (p *sequenceParser) extractNext(length int) (string, error) {
	if p.separator != "" {
		parts := strings.Split(p.sequence[p.pos:], p.separator)
		if len(parts) == 0 {
			return "", ErrUnexpectedEndOfSequence
		}
		result := parts[0]
		p.pos += len(result)
		if p.pos < len(p.sequence) && p.sequence[p.pos:p.pos+len(p.separator)] == p.separator {
			p.pos += len(p.separator)
		}
		return result, nil
	}
	if p.pos+length > len(p.sequence) {
		return "", fmt.Errorf(
			"sequence too short: expected at least %d chars from position %d",
			length,
			p.pos,
		)
	}
	result := p.sequence[p.pos : p.pos+length]
	p.pos += length
	return result, nil
}

func (g *generator) parseCustomFormat(
	sequence string,
	_ *Format,
) (*SequenceComponents, error) {
	// For custom formats, we can't reliably parse without knowing the exact pattern
	// This is a limitation but provides a basic implementation
	return &SequenceComponents{
		Original: sequence,
	}, nil
}
