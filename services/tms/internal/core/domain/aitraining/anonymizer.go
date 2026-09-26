package aitraining

import (
	"errors"
	"math/rand/v2"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/piiscrub"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

var ErrResidualIdentifier = errors.New("a known identifier survived anonymization")

const (
	anonymousFileStem   = "document"
	maxExtensionRunes   = 5
	minMoneyFactorBasis = 8500
	moneyFactorSpread   = 3001
	moneyFactorScale    = 10000
	neutralBandLow      = 9900
	neutralBandHigh     = 10100
	maxSurrogateTries   = 8
	sentinelOpen        = ''
	sentinelClose       = ''
	sentinelDigitBase   = ''
	dateDigits          = 8
)

var (
	unitSuffixPattern = regexp.MustCompile(
		`(?i)^\d[\d,]*(?:lbs?|kgs?|ft|pcs?|plts?|mi|miles|hrs?|am|pm|st|nd|rd|th|in|oz|gal|cwt)$`,
	)
	extensionPattern = regexp.MustCompile(`^[a-z0-9]{1,5}$`)
)

type Source struct {
	FileName   string
	Pages      []ExamplePage
	Target     *aicorrection.Snapshot
	Prediction *aicorrection.Snapshot
	Issuer     string
}

type Anonymized struct {
	FileName   string
	Pages      []ExamplePage
	Target     *ExampleSnapshot
	Prediction *ExampleSnapshot
}

func Anonymize(src *Source, identity *Identity, rng *rand.Rand) (*Anonymized, error) {
	set := newKnownSet()
	if identity != nil {
		for _, value := range identity.known {
			set.add(value)
		}
	}
	set.addSnapshot(src.Target)
	set.addSnapshot(src.Prediction)
	set.add(partyKnown(src.Issuer))
	known := set.sorted()

	p := newPseudonymizer(rng)
	out := &Anonymized{
		FileName: anonymousFileName(src.FileName),
		Pages:    make([]ExamplePage, 0, len(src.Pages)),
	}
	for i := range src.Pages {
		out.Pages = append(out.Pages, ExamplePage{
			Number: src.Pages[i].Number,
			Text:   p.text(src.Pages[i].Text, known),
		})
	}
	out.Target = p.snapshot(src.Target, known)
	out.Prediction = p.snapshot(src.Prediction, known)

	if residual(out, known) {
		return nil, ErrResidualIdentifier
	}

	return out, nil
}

type pseudonymizer struct {
	rng       *rand.Rand
	factor    decimal.Decimal
	memo      map[category]map[string]string
	sentinels []string
}

func newPseudonymizer(rng *rand.Rand) *pseudonymizer {
	return &pseudonymizer{
		rng:    rng,
		factor: moneyFactor(rng),
		memo:   map[category]map[string]string{},
	}
}

func moneyFactor(rng *rand.Rand) decimal.Decimal {
	basis := minMoneyFactorBasis + rng.IntN(moneyFactorSpread)
	for basis > neutralBandLow && basis < neutralBandHigh {
		basis = minMoneyFactorBasis + rng.IntN(moneyFactorSpread)
	}

	return decimal.NewFromInt(int64(basis)).Div(decimal.NewFromInt(moneyFactorScale))
}

func (p *pseudonymizer) surrogate(c category, key string, generate func() string) string {
	values, ok := p.memo[c]
	if !ok {
		values = map[string]string{}
		p.memo[c] = values
	}
	if existing, found := values[key]; found {
		return existing
	}

	value := generate()
	for range maxSurrogateTries {
		if stringutils.NormalizeName(value) != stringutils.NormalizeName(key) {
			break
		}
		value = generate()
	}
	values[key] = value

	return value
}

func (p *pseudonymizer) hold(value string) string {
	index := len(p.sentinels)
	p.sentinels = append(p.sentinels, value)

	var b strings.Builder
	b.WriteRune(sentinelOpen)
	for _, digit := range strconv.Itoa(index) {
		b.WriteRune(sentinelDigitBase + (digit - '0'))
	}
	b.WriteRune(sentinelClose)

	return b.String()
}

func (p *pseudonymizer) expand(text string) string {
	if !strings.ContainsRune(text, sentinelOpen) {
		return text
	}

	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if r != sentinelOpen {
			b.WriteRune(r)
			i += size
			continue
		}
		i += size
		index := 0
		for i < len(text) {
			digit, digitSize := utf8.DecodeRuneInString(text[i:])
			i += digitSize
			if digit == sentinelClose {
				break
			}
			index = index*10 + int(digit-sentinelDigitBase)
		}
		if index < len(p.sentinels) {
			b.WriteString(p.sentinels[index])
		}
	}

	return b.String()
}

func (p *pseudonymizer) text(text string, known []*knownValue) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	for _, rule := range addressRules {
		text = rule(p, text)
	}
	for _, value := range known {
		text = p.applyKnown(text, value)
	}
	for _, rule := range genericRules {
		text = rule(p, text)
	}

	return piiscrub.Scrub(p.expand(text))
}

func (p *pseudonymizer) applyKnown(text string, value *knownValue) string {
	accept := bounded
	if value.category == categoryMoney {
		accept = moneyBounded
	}

	return replaceMatches(text, value.pattern, accept, func(match []int) string {
		original := text[match[0]:match[1]]
		switch value.category {
		case categoryParty:
			base := text[match[2]:match[3]]
			name := p.surrogate(categoryParty, value.key, func() string { return companySurrogate(p.rng) })
			return p.hold(styleLike(name, base) + text[match[3]:match[1]])
		case categoryPerson:
			return p.hold(styleLike(p.surrogate(categoryPerson, value.key, p.person), original))
		case categoryStreet:
			return p.hold(styleLike(p.surrogate(categoryStreet, value.key, p.street), original))
		case categoryCity:
			return p.hold(styleLike(p.surrogate(categoryCity, value.key, p.city), original))
		case categoryPostal:
			return p.hold(p.postal(value.key, text[match[3]:match[1]]))
		case categoryIdentifier:
			return p.hold(p.identifier(value.key, original))
		case categoryMoney:
			return p.hold(p.money(original, value.amount))
		default:
			return original
		}
	})
}

func (p *pseudonymizer) person() string { return personSurrogate(p.rng) }

func (p *pseudonymizer) street() string { return streetSurrogate(p.rng) }

func (p *pseudonymizer) city() string { return citySurrogate(p.rng) }

func (p *pseudonymizer) postal(five, extension string) string {
	value := p.surrogate(categoryPostal, five, func() string { return postalSurrogate(p.rng) })
	if extension == "" {
		return value
	}

	return value + shapeSurrogate(p.rng, extension)
}

func (p *pseudonymizer) identifier(key, original string) string {
	value := p.surrogate(categoryIdentifier, key, func() string { return shapeSurrogate(p.rng, key) })

	return reshape(original, value)
}

func (p *pseudonymizer) scale(amount decimal.Decimal) decimal.Decimal {
	return amount.Mul(p.factor).Round(2)
}

func (p *pseudonymizer) money(original string, amount decimal.Decimal) string {
	firstDigit := strings.IndexFunc(original, isDigitRune)
	if firstDigit < 0 {
		return original
	}
	prefix := original[:firstDigit]
	number := original[firstDigit:]

	places := int32(0)
	if dot := strings.LastIndexByte(number, '.'); dot >= 0 {
		places = int32(len(number) - dot - 1)
	}
	scaled := p.scale(amount)
	if places == 0 && !scaled.Equal(scaled.Truncate(0)) {
		places = 2
	}
	rendered := scaled.StringFixed(places)
	if strings.ContainsRune(number, ',') {
		whole, fraction, hasFraction := strings.Cut(rendered, ".")
		parsed, err := strconv.ParseInt(whole, 10, 64)
		if err == nil {
			rendered = intutils.FormatWithCommas(parsed)
			if hasFraction {
				rendered += "." + fraction
			}
		}
	}

	return prefix + rendered
}

func (p *pseudonymizer) snapshot(snapshot *aicorrection.Snapshot, known []*knownValue) *ExampleSnapshot {
	out := &ExampleSnapshot{Fields: map[string]string{}, Stops: []ExampleStop{}}
	if snapshot == nil {
		return out
	}
	for key, value := range snapshot.Fields {
		if text := p.text(value, known); text != "" {
			out.Fields[key] = text
		}
	}
	for i := range snapshot.Stops {
		stop := &snapshot.Stops[i]
		out.Stops = append(out.Stops, ExampleStop{
			Role:                stop.Role,
			Sequence:            stop.Sequence,
			Name:                p.text(stop.Name, known),
			AddressLine1:        p.text(stop.AddressLine1, known),
			AddressLine2:        p.text(stop.AddressLine2, known),
			City:                p.text(stop.City, known),
			State:               stop.State,
			PostalCode:          p.text(stop.PostalCode, known),
			Date:                stop.Date,
			TimeWindow:          p.text(stop.TimeWindow, known),
			AppointmentRequired: stop.AppointmentRequired,
		})
	}

	return out
}

func residual(out *Anonymized, known []*knownValue) bool {
	var b strings.Builder
	b.WriteString(out.FileName)
	for i := range out.Pages {
		b.WriteByte('\n')
		b.WriteString(out.Pages[i].Text)
	}
	writeSnapshot(&b, out.Target)
	writeSnapshot(&b, out.Prediction)
	joined := b.String()

	tokens := " " + stringutils.NormalizeName(joined) + " "
	stream := stringutils.NormalizeIdentifier(joined)
	for _, value := range known {
		if value.check != "" && strings.Contains(tokens, " "+value.check+" ") {
			return true
		}
		if value.category == categoryIdentifier &&
			utf8.RuneCountInString(value.key) >= minStreamIdentifier &&
			strings.Contains(stream, value.key) {
			return true
		}
	}

	return false
}

func writeSnapshot(b *strings.Builder, snapshot *ExampleSnapshot) {
	if snapshot == nil {
		return
	}
	for _, value := range snapshot.Fields {
		b.WriteByte('\n')
		b.WriteString(value)
	}
	for i := range snapshot.Stops {
		stop := &snapshot.Stops[i]
		for _, value := range []string{
			stop.Name, stop.AddressLine1, stop.AddressLine2, stop.City, stop.PostalCode, stop.TimeWindow,
		} {
			b.WriteByte('\n')
			b.WriteString(value)
		}
	}
}

func anonymousFileName(name string) string {
	extension := strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(name)), "."))
	if extension == "" || utf8.RuneCountInString(extension) > maxExtensionRunes ||
		!extensionPattern.MatchString(extension) {
		return anonymousFileStem
	}

	return anonymousFileStem + "." + extension
}

func replaceMatches(
	text string,
	pattern *regexp.Regexp,
	accept func(text string, start, end int) bool,
	render func(match []int) string,
) string {
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var b strings.Builder
	cursor := 0
	for _, match := range matches {
		if accept != nil && !accept(text, match[0], match[1]) {
			continue
		}
		if b.Len() == 0 {
			b.Grow(len(text))
		}
		b.WriteString(text[cursor:match[0]])
		b.WriteString(render(match))
		cursor = match[1]
	}
	if cursor == 0 {
		return text
	}
	b.WriteString(text[cursor:])

	return b.String()
}

func bounded(text string, start, end int) bool {
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); isWordRune(r) {
			return false
		}
	}
	if end < len(text) {
		if r, _ := utf8.DecodeRuneInString(text[end:]); isWordRune(r) {
			return false
		}
	}

	return true
}

func moneyBounded(text string, start, end int) bool {
	if !bounded(text, start, end) {
		return false
	}
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); r == '.' || r == ',' {
			if start > 1 {
				if prior, _ := utf8.DecodeLastRuneInString(text[:start-1]); isDigitRune(prior) {
					return false
				}
			}
		}
	}
	if end+1 < len(text) && (text[end] == '.' || text[end] == ',') && isDigitRune(rune(text[end+1])) {
		return false
	}

	return true
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

func looksLikeCompactDate(digits string) bool {
	if len(digits) != dateDigits {
		return false
	}
	year, month, day := digits[:4], digits[4:6], digits[6:]
	if validDate(year, month, day) {
		return true
	}

	return validDate(digits[4:], digits[:2], digits[2:4])
}

func validDate(year, month, day string) bool {
	y, errY := strconv.Atoi(year)
	m, errM := strconv.Atoi(month)
	d, errD := strconv.Atoi(day)
	if errY != nil || errM != nil || errD != nil {
		return false
	}

	return y >= 1990 && y <= 2100 && m >= 1 && m <= 12 && d >= 1 && d <= 31
}
