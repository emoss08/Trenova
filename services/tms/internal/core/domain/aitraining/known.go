package aitraining

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

type category int

const (
	categoryParty category = iota
	categoryPerson
	categoryStreet
	categoryCity
	categoryPostal
	categoryIdentifier
	categoryMoney
	categoryEmail
	categoryPhone
)

const (
	tokenSeparator         = `[^\p{L}\p{N}\x{E000}-\x{E0FF}]{1,3}`
	identifierSeparator    = `[\s\-./#]?`
	minNameRunes           = 3
	minCheckedRunes        = 4
	minIdentifierRunes     = 4
	minNumericIdentifier   = 5
	minStreamIdentifier    = 6
	postalDigits           = 5
	legalSuffixAlternation = `incorporated|corporation|company|limited|pllc|corp|inc|llc|ltd|llp|plc|lp|co|l\.?\s?l\.?\s?c`
)

var (
	partyFieldKeys = map[string]struct{}{
		aicorrection.FieldShipper:   {},
		aicorrection.FieldConsignee: {},
		"billTo":                    {},
		"carrierName":               {},
		"broker":                    {},
		"customer":                  {},
	}
	personFieldKeys = map[string]struct{}{
		"carrierContact": {},
		"contact":        {},
		"dispatcher":     {},
	}
	identifierFieldKeys = map[string]struct{}{
		aicorrection.FieldReference: {},
		"reference":                 {},
		"loadNumber":                {},
		"bol":                       {},
		"poNumber":                  {},
		"proNumber":                 {},
		"pickupNumber":              {},
		"deliveryNumber":            {},
		"appointmentNumber":         {},
		"containerNumber":           {},
		"trailerNumber":             {},
		"tractorNumber":             {},
		"sealNumber":                {},
		"scac":                      {},
	}
	moneyFieldKeys = map[string]struct{}{
		aicorrection.FieldRate: {},
		"fuelSurcharge":        {},
		"lineHaul":             {},
		"totalRate":            {},
	}
	streetEquivalents = [][]string{
		{"st", "street"},
		{"ave", "av", "avenue"},
		{"rd", "road"},
		{"blvd", "boulevard"},
		{"dr", "drive"},
		{"ln", "lane"},
		{"hwy", "highway"},
		{"pkwy", "parkway"},
		{"ct", "court"},
		{"pl", "place"},
		{"cir", "circle"},
		{"trl", "trail"},
		{"ter", "terrace"},
		{"ste", "suite"},
		{"n", "north"},
		{"s", "south"},
		{"e", "east"},
		{"w", "west"},
	}
	streetTokenPatterns = buildStreetTokenPatterns()
)

type knownValue struct {
	category category
	key      string
	check    string
	pattern  *regexp.Regexp
	amount   decimal.Decimal
}

type IdentityValues struct {
	Parties     []string
	People      []string
	Identifiers []string
	Streets     []string
	Cities      []string
	PostalCodes []string
}

type Identity struct {
	known []*knownValue
}

func NewIdentity(values *IdentityValues) *Identity {
	set := newKnownSet()
	if values != nil {
		set.addEach(values.Parties, partyKnown)
		set.addEach(values.People, personKnown)
		set.addEach(values.Identifiers, identifierKnown)
		set.addEach(values.Streets, streetKnown)
		set.addEach(values.Cities, cityKnown)
		set.addEach(values.PostalCodes, postalKnown)
	}

	return &Identity{known: set.values}
}

func (i *Identity) Size() int {
	if i == nil {
		return 0
	}

	return len(i.known)
}

type knownSet struct {
	seen   map[category]map[string]struct{}
	values []*knownValue
}

func newKnownSet() *knownSet {
	return &knownSet{seen: map[category]map[string]struct{}{}}
}

func (s *knownSet) add(value *knownValue) {
	if value == nil {
		return
	}
	keys, ok := s.seen[value.category]
	if !ok {
		keys = map[string]struct{}{}
		s.seen[value.category] = keys
	}
	if _, dup := keys[value.key]; dup {
		return
	}
	keys[value.key] = struct{}{}
	s.values = append(s.values, value)
}

func (s *knownSet) addEach(values []string, compile func(string) *knownValue) {
	for _, value := range values {
		s.add(compile(value))
	}
}

func (s *knownSet) addSnapshot(snapshot *aicorrection.Snapshot) {
	if snapshot == nil {
		return
	}
	for key, value := range snapshot.Fields {
		s.add(fieldKnown(key, value))
	}
	for i := range snapshot.Stops {
		stop := &snapshot.Stops[i]
		s.add(partyKnown(stop.Name))
		s.add(streetKnown(stop.AddressLine1))
		s.add(streetKnown(stop.AddressLine2))
		s.add(cityKnown(stop.City))
		s.add(postalKnown(stop.PostalCode))
	}
}

func (s *knownSet) sorted() []*knownValue {
	out := slices.Clone(s.values)
	slices.SortStableFunc(out, func(a, b *knownValue) int {
		if c := cmp.Compare(len(b.pattern.String()), len(a.pattern.String())); c != 0 {
			return c
		}
		return cmp.Compare(a.category, b.category)
	})

	return out
}

func fieldKnown(key, value string) *knownValue {
	if _, ok := partyFieldKeys[key]; ok {
		return partyKnown(value)
	}
	if _, ok := personFieldKeys[key]; ok {
		return personKnown(value)
	}
	if _, ok := identifierFieldKeys[key]; ok {
		return identifierKnown(value)
	}
	if _, ok := moneyFieldKeys[key]; ok {
		return moneyKnown(value)
	}

	return nil
}

func partyKnown(value string) *knownValue {
	base := stringutils.NormalizeCompanyName(value)
	if alnumRunes(base) < minNameRunes {
		return nil
	}
	tokens := strings.Fields(base)

	return &knownValue{
		category: categoryParty,
		key:      base,
		check:    checked(base),
		pattern: regexp.MustCompile(
			`(?i)(` + joinTokens(tokens, quoteToken) + `)(?:` + tokenSeparator +
				`(?:` + legalSuffixAlternation + `)\b\.?)?`,
		),
	}
}

func personKnown(value string) *knownValue {
	name := stringutils.NormalizeName(value)
	tokens := strings.Fields(name)
	if len(tokens) < 2 || alnumRunes(name) < minCheckedRunes {
		return nil
	}

	return &knownValue{
		category: categoryPerson,
		key:      name,
		check:    name,
		pattern:  regexp.MustCompile(`(?i)` + joinTokens(tokens, quoteToken)),
	}
}

func streetKnown(value string) *knownValue {
	name := stringutils.NormalizeName(value)
	tokens := strings.Fields(name)
	if len(tokens) < 2 || alnumRunes(name) < postalDigits {
		return nil
	}

	return &knownValue{
		category: categoryStreet,
		key:      canonicalStreet(tokens),
		check:    name,
		pattern:  regexp.MustCompile(`(?i)` + joinTokens(tokens, streetToken)),
	}
}

func cityKnown(value string) *knownValue {
	name := stringutils.NormalizeName(value)
	if alnumRunes(name) < minNameRunes {
		return nil
	}

	return &knownValue{
		category: categoryCity,
		key:      name,
		check:    checked(name),
		pattern:  regexp.MustCompile(`(?i)` + joinTokens(strings.Fields(name), quoteToken)),
	}
}

func postalKnown(value string) *knownValue {
	digits := stringutils.DigitsOnly(value)
	if len(digits) < postalDigits {
		return nil
	}
	five := digits[:postalDigits]

	return &knownValue{
		category: categoryPostal,
		key:      five,
		check:    five,
		pattern:  regexp.MustCompile(`(` + five + `)(?:-\d{4})?`),
	}
}

func identifierKnown(value string) *knownValue {
	key := stringutils.NormalizeIdentifier(value)
	length := utf8.RuneCountInString(key)
	if length < minIdentifierRunes {
		return nil
	}
	if stringutils.DigitsOnly(key) == key && length < minNumericIdentifier {
		return nil
	}

	parts := make([]string, 0, length)
	for _, r := range key {
		parts = append(parts, regexp.QuoteMeta(string(r)))
	}
	check := ""
	if length >= minCheckedRunes && strings.ContainsFunc(key, isLetterRune) {
		check = strings.ToLower(key)
	}

	return &knownValue{
		category: categoryIdentifier,
		key:      key,
		check:    check,
		pattern:  regexp.MustCompile(`(?i)` + strings.Join(parts, identifierSeparator)),
	}
}

func moneyKnown(value string) *knownValue {
	parsed, err := decimalutils.ParseMoneyText(strings.TrimFunc(value, isNotDigitRune))
	if err != nil || !parsed.Valid || !parsed.Decimal.IsPositive() {
		return nil
	}
	amount := parsed.Decimal.Round(2)
	whole := amount.Truncate(0)
	integer := whole.String()
	grouped := intutils.FormatWithCommas(whole.IntPart())

	alternatives := regexp.QuoteMeta(integer)
	if grouped != integer {
		alternatives = regexp.QuoteMeta(grouped) + "|" + alternatives
	}
	cents := `(?:\.00)?`
	if !amount.Equal(whole) {
		cents = fmt.Sprintf(`\.%02d`, amount.Sub(whole).Shift(2).IntPart())
	}

	return &knownValue{
		category: categoryMoney,
		key:      amount.StringFixed(2),
		amount:   amount,
		pattern:  regexp.MustCompile(`(?i)(?:\$\s?|usd\s?)?(?:` + alternatives + `)` + cents),
	}
}

func joinTokens(tokens []string, render func(string) string) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, render(token))
	}

	return strings.Join(parts, tokenSeparator)
}

func quoteToken(token string) string {
	return regexp.QuoteMeta(token)
}

func streetToken(token string) string {
	if pattern, ok := streetTokenPatterns[token]; ok {
		return pattern
	}

	return regexp.QuoteMeta(token)
}

func canonicalStreet(tokens []string) string {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, canonicalStreetToken(token))
	}

	return strings.Join(out, " ")
}

func canonicalStreetToken(token string) string {
	for _, group := range streetEquivalents {
		if slices.Contains(group, token) {
			return group[0]
		}
	}

	return token
}

func buildStreetTokenPatterns() map[string]string {
	patterns := make(map[string]string, len(streetEquivalents)*3)
	for _, group := range streetEquivalents {
		quoted := make([]string, 0, len(group))
		for _, variant := range slices.Sorted(slices.Values(group)) {
			quoted = append(quoted, regexp.QuoteMeta(variant))
		}
		slices.SortStableFunc(quoted, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
		pattern := `(?:` + strings.Join(quoted, "|") + `)\.?`
		for _, variant := range group {
			patterns[variant] = pattern
		}
	}

	return patterns
}

func checked(normalized string) string {
	if alnumRunes(normalized) < minCheckedRunes {
		return ""
	}

	return normalized
}

func alnumRunes(value string) int {
	count := 0
	for _, r := range value {
		if isWordRune(r) {
			count++
		}
	}

	return count
}

func isLetterRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
