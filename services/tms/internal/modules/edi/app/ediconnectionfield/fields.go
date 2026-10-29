package ediconnectionfield

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
)

type Scope int

const (
	ScopePartner Scope = iota
	ScopeTransport
	ScopeInterchange
	ScopeSet
	ScopeControlNumber
)

type normalizer func(string) (string, error)

type quoteMatcher func(value, quote string) bool

type Spec struct {
	Key       string
	Label     string
	Scope     Scope
	ConfigKey string
	methods   []edi.ConnectionMethod
	Required  bool
	Normalize normalizer
	spelled   map[string][]string
	matches   quoteMatcher
	Fallback  string
}

const (
	PartnerCode         = "partner.code"
	PartnerName         = "partner.name"
	PartnerContactName  = "partner.contactName"
	PartnerContactEmail = "partner.contactEmail"
	PartnerContactPhone = "partner.contactPhone"
	PartnerTimezone     = "partner.timezone"
	PartnerCountry      = "partner.country"

	TransportMethod      = "transport.method"
	TransportEnvironment = "transport.environment"

	PartnerQualifier = "isa.partnerQualifier"
	PartnerID        = "isa.partnerId"
	OurQualifier     = "isa.ourQualifier"
	OurID            = "isa.ourId"
	PartnerGSID      = "gs.partnerId"
	OurGSID          = "gs.ourId"
	UsageIndicator   = "isa.usageIndicator"
	X12Version       = "x12.version"
	ElementSep       = "separators.element"
	SegmentSep       = "separators.segment"
	ComponentSep     = "separators.component"
	RepetitionSep    = "separators.repetition"

	SetAttrSupported        = "supported"
	SetAttrVersion          = "version"
	SetAttrAck              = "ack"
	SetAttrAckWindowMinutes = "ackWindowMinutes"

	controlNumberAttrStart = "start"

	AckNone = "None"

	ValueTrue       = "true"
	ValueFalse      = "false"
	ValueTest       = "test"
	ValueProduction = "production"
	valueSync       = "sync"
	valueAsync      = "async"
	valueNone       = "none"
	valuePassword   = "password"
	valuePrivateKey = "privateKey"
	valueAS2        = string(edi.ConnectionMethodAS2)
	valueSFTP       = string(edi.ConnectionMethodSFTP)
	valueVAN        = string(edi.ConnectionMethodVAN)
)

var (
	errBlank        = errors.New("the value is blank")
	isaQualifierRe  = regexp.MustCompile(`^[A-Z0-9]{2}$`)
	versionDigitsRe = regexp.MustCompile(`^\d{4,6}$`)
	functionalIDRe  = regexp.MustCompile(`^[A-Z]{2}$`)
	numberRe        = regexp.MustCompile(`\d+`)
	durationRe      = regexp.MustCompile(
		`(?i)(\d+(?:\.\d+)?)\s*(minutes?|mins?|hours?|hrs?|days?|business days?)`,
	)
	countryRe       = regexp.MustCompile(`^[A-Z]{2}$`)
	externalMethods = []edi.ConnectionMethod{
		edi.ConnectionMethodAS2,
		edi.ConnectionMethodSFTP,
		edi.ConnectionMethodVAN,
	}
	fileMethods   = []edi.ConnectionMethod{edi.ConnectionMethodSFTP, edi.ConnectionMethodVAN}
	as2Only       = []edi.ConnectionMethod{edi.ConnectionMethodAS2}
	vanOnly       = []edi.ConnectionMethod{edi.ConnectionMethodVAN}
	SupportedSets = []edidocument.TransactionSet{
		edidocument.TransactionSet204,
		edidocument.TransactionSet210,
		edidocument.TransactionSet214,
		edidocument.TransactionSet990,
	}
	Directions = []edidocument.DocumentDirection{
		edidocument.DocumentDirectionInbound,
		edidocument.DocumentDirectionOutbound,
	}
	ControlNumberKinds = []edidocument.ControlNumberKind{
		edidocument.ControlNumberKindInterchange,
		edidocument.ControlNumberKindGroup,
		edidocument.ControlNumberKindTransaction,
	}
)

var secretWords = []string{
	valuePassword, "passphrase", "privatekey", "private_key", "secret", "token",
	"credential", "certificate", "knownhostkey", "hostkey", "apikey", "authorization",
	"security",
}

func isSecretField(key string) bool {
	compact := strings.ToLower(strings.NewReplacer(".", "", "-", "", " ", "").Replace(key))
	for _, word := range secretWords {
		if strings.Contains(compact, word) {
			return true
		}
	}
	return false
}

func text(maxLength int) normalizer {
	return func(value string) (string, error) {
		value = strings.Join(strings.Fields(value), " ")
		if value == "" {
			return "", errBlank
		}
		if len(value) > maxLength {
			return "", fmt.Errorf("must be at most %d characters", maxLength)
		}
		return value, nil
	}
}

func choice(allowed map[string]string) normalizer {
	return func(value string) (string, error) {
		key := strings.ToLower(strings.Join(strings.Fields(value), " "))
		if key == "" {
			return "", errBlank
		}
		if normalized, ok := allowed[key]; ok {
			return normalized, nil
		}
		options := make([]string, 0, len(allowed))
		for _, normalized := range allowed {
			if !slices.Contains(options, normalized) {
				options = append(options, normalized)
			}
		}
		slices.Sort(options)
		return "", fmt.Errorf("must be one of %s", strings.Join(options, ", "))
	}
}

type spelledMatch struct {
	value string
	start int
	end   int
}

func spelledOut(normalize normalizer, spelled map[string][]string) normalizer {
	return func(value string) (string, error) {
		normalized, err := normalize(value)
		if err == nil || errors.Is(err, errBlank) {
			return normalized, err
		}
		target, ok := spelledValue(value, spelled)
		if !ok {
			return "", err
		}
		return normalize(target)
	}
}

func spelledValue(value string, spelled map[string][]string) (string, bool) {
	haystack := NormalizeForMatch(value)
	matches := make([]spelledMatch, 0, 4)
	for target, phrases := range spelled {
		matches = appendSpelled(matches, haystack, target, target)
		for _, phrase := range phrases {
			matches = appendSpelled(matches, haystack, target, phrase)
		}
	}

	found := ""
	for idx := range matches {
		if insideLonger(matches[idx], matches) {
			continue
		}
		if found != "" && found != matches[idx].value {
			return "", false
		}
		found = matches[idx].value
	}
	return found, found != ""
}

func appendSpelled(matches []spelledMatch, haystack, target, phrase string) []spelledMatch {
	needle := NormalizeForMatch(phrase)
	if needle == "" {
		return matches
	}
	for start := 0; ; {
		index := tokenIndex(haystack, needle, start)
		if index < 0 {
			return matches
		}
		matches = append(matches, spelledMatch{
			value: target,
			start: index,
			end:   index + len(needle),
		})
		start = index + 1
	}
}

func insideLonger(match spelledMatch, matches []spelledMatch) bool {
	for _, other := range matches {
		if other.start <= match.start && other.end >= match.end &&
			other.end-other.start > match.end-match.start {
			return true
		}
	}
	return false
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value {
		return "", errors.New("must be an email address")
	}
	return value, nil
}

func normalizeTimezone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if _, err := time.LoadLocation(value); err != nil || !strings.Contains(value, "/") {
		return "", errors.New("must be an IANA time zone such as America/Chicago")
	}
	return value, nil
}

func normalizeCountry(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !countryRe.MatchString(value) {
		return "", errors.New("must be a two-letter country code")
	}
	return value, nil
}

func normalizeQualifier(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if !isaQualifierRe.MatchString(value) {
		return "", errors.New("must be a two-character ISA qualifier")
	}
	return value, nil
}

func normalizeISAID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if len(value) > 15 {
		return "", errors.New("must be at most 15 characters")
	}
	return value, nil
}

func normalizeGSID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || len(value) > 15 {
		return "", errors.New("must be 2 to 15 characters")
	}
	return value, nil
}

func normalizeVersion(value string) (string, error) {
	digits := strings.TrimSpace(value)
	if !versionDigitsRe.MatchString(digits) {
		return "", errors.New("must be an X12 version such as 004010")
	}
	switch len(digits) {
	case 4:
		digits = "00" + digits
	case 5:
		digits += "0"
	}
	return digits, nil
}

func versionCore(version string) string {
	normalized, err := normalizeVersion(version)
	if err != nil {
		return ""
	}
	return strings.TrimLeft(normalized, "0")[:3]
}

func matchesVersion(value, quote string) bool {
	core := versionCore(value)
	if core == "" {
		return false
	}
	for _, found := range numberRe.FindAllString(quote, -1) {
		if versionDigitsRe.MatchString(found) && versionCore(found) == core {
			return true
		}
	}
	return false
}

var separatorNames = map[string]string{
	"asterisk": "*", "star": "*", "tilde": "~", "greater than": ">", "greater-than": ">",
	"caret": "^", "pipe": "|", "vertical bar": "|", "colon": ":", "plus": "+",
	"exclamation": "!", "exclamation mark": "!", "back tick": "`", "backtick": "`",
	"new line": "\n", "newline": "\n", "line feed": "\n", "carriage return": "\n",
	"crlf": "\n", "apostrophe": "'", "single quote": "'", "comma": ",",
}

func normalizeSeparator(value string) (string, error) {
	if value == "\n" || value == "\r\n" {
		return "\n", nil
	}
	trimmed := strings.TrimSpace(value)
	if named, ok := separatorNames[strings.ToLower(trimmed)]; ok {
		return named, nil
	}
	if len([]rune(trimmed)) != 1 {
		return "", errors.New("must be a single character")
	}
	return trimmed, nil
}

func spelledSeparators() map[string][]string {
	spelled := make(map[string][]string, len(separatorNames))
	for name, char := range separatorNames {
		spelled[char] = append(spelled[char], name)
	}
	return spelled
}

func normalizeURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", errors.New("must be an absolute http or https URL")
	}
	return value, nil
}

func normalizePort(value string) (string, error) {
	value = strings.TrimSpace(value)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return "", errors.New("must be a port number between 1 and 65535")
	}
	return strconv.Itoa(port), nil
}

func normalizePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errBlank
	}
	if len(value) > 255 || strings.Contains(value, "..") {
		return "", errors.New(
			"must be a directory path of at most 255 characters, without a parent reference",
		)
	}
	return value, nil
}

func normalizeMinutes(value string) (string, error) {
	value = strings.TrimSpace(value)
	if minutes, err := strconv.Atoi(value); err == nil {
		if minutes < 1 || minutes > 60*24*30 {
			return "", errors.New("must be between 1 minute and 30 days")
		}
		return strconv.Itoa(minutes), nil
	}
	durations := durationMinutes(value)
	if len(durations) != 1 {
		return "", errors.New("must be a number of minutes")
	}
	return normalizeMinutes(strconv.Itoa(durations[0]))
}

func durationMinutes(text string) []int {
	matches := durationRe.FindAllStringSubmatch(text, -1)
	out := make([]int, 0, len(matches))
	for _, match := range matches {
		amount, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		unit := strings.ToLower(match[2])
		switch {
		case strings.HasPrefix(unit, "min"):
		case strings.HasPrefix(unit, "h"):
			amount *= 60
		case strings.HasPrefix(unit, "business"):
			amount *= 60 * 24
		case strings.HasPrefix(unit, "day"):
			amount *= 60 * 24
		}
		out = append(out, int(amount))
	}
	return out
}

func matchesMinutes(value, quote string) bool {
	minutes, err := strconv.Atoi(value)
	if err != nil {
		return false
	}
	for _, found := range durationMinutes(quote) {
		if found == minutes {
			return true
		}
	}
	return false
}

func normalizeControlNumber(value string) (string, error) {
	value = strings.TrimLeft(strings.TrimSpace(value), "0")
	if value == "" {
		value = "0"
	}
	number, err := strconv.ParseInt(value, 10, 64)
	if err != nil || number < 1 || number > 999999999 {
		return "", errors.New("must be a whole number between 1 and 999999999")
	}
	return strconv.FormatInt(number, 10), nil
}

func matchesNumber(value, quote string) bool {
	for _, found := range numberRe.FindAllString(quote, -1) {
		if strings.TrimLeft(found, "0") == strings.TrimLeft(value, "0") {
			return true
		}
	}
	return false
}

func anyQuote(string, string) bool { return true }

func as2Algorithms(values ...string) normalizer {
	allowed := make(map[string]string, len(values))
	for _, value := range values {
		allowed[value] = value
	}
	return choice(allowed)
}
