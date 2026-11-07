package sensitivetext

import (
	"regexp"
	"slices"
	"strings"
)

const stars = "****"

type namedPattern struct {
	name  string
	regex *regexp.Regexp
}

var valuePatterns = compile(map[string]string{
	"ssn":            `\b\d{3}-\d{2}-\d{4}\b|\b\d{9}\b`,
	"creditCard":     `\b(?:\d[ -]*?){13,19}\b`,
	"email":          `\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`,
	"phone":          `\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`,
	"apiKey":         `\b(?i)(api[_-]?key|apikey|api[_-]?secret|access[_-]?token|auth[_-]?token|bearer)\s*[:=]\s*["']?[\w\-]{20,}["']?\b|^[A-Za-z0-9]{20,}$`,
	"googleApiKey":   `\bAIza[0-9A-Za-z\-_]{35}\b`,
	"awsKey":         `\b(?:AKIA|A3T|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[0-9A-Z]{16}\b`,
	"jwt":            `\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`,
	"dbConnection":   `\b(?i)(mongodb|postgres|postgresql|mysql|redis|mssql|oracle):\/\/[^\s]+\b`,
	"privateKey":     `-----BEGIN\s+(?:RSA\s+)?PRIVATE\s+KEY-----[\s\S]+?-----END\s+(?:RSA\s+)?PRIVATE\s+KEY-----`,
	"ipAddress":      `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
	"bankAccount":    `\b\d{9}\s*\d{1,17}\b`,
	"driversLicense": `\b(?i)(license|licence|dl|driver)\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{5,20}\b`,
	"dateOfBirth":    `\b(?:0[1-9]|1[0-2])[-/](?:0[1-9]|[12][0-9]|3[01])[-/](?:19|20)\d{2}\b`,
	"taxId":          `\b\d{2}-\d{7}\b`,
	"passport":       `\b(?i)passport\s*(?:number|no|#)?\s*[:=]?\s*[A-Z0-9]{6,20}\b`,
})

var fieldPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)password`),
	regexp.MustCompile(`(?i)secret`),
	regexp.MustCompile(`(?i)token`),
	regexp.MustCompile(`(?i)api[_-]?key`),
	regexp.MustCompile(`(?i)apikey`),
	regexp.MustCompile(`(?i)private[_-]?key`),
	regexp.MustCompile(`(?i)ssn|social[_-]?security`),
	regexp.MustCompile(`(?i)credit[_-]?card`),
	regexp.MustCompile(`(?i)bank[_-]?account`),
	regexp.MustCompile(`(?i)routing[_-]?number`),
	regexp.MustCompile(`(?i)license[_-]?number`),
	regexp.MustCompile(`(?i)passport`),
	regexp.MustCompile(`(?i)tax[_-]?id`),
	regexp.MustCompile(`(?i)date[_-]?of[_-]?birth|dob`),
	regexp.MustCompile(`(?i)salary|wage|income`),
	regexp.MustCompile(`(?i)medical|health`),
}

var (
	opaqueToken = regexp.MustCompile(`^[A-Za-z0-9_\-]{20,}$`)
	recordID    = regexp.MustCompile(`^[a-z]{1,12}_[0-9A-HJKMNP-TV-Z]{26}$`)
)

func compile(patterns map[string]string) []namedPattern {
	out := make([]namedPattern, 0, len(patterns))
	for name, pattern := range patterns {
		out = append(out, namedPattern{name: name, regex: regexp.MustCompile(pattern)})
	}
	slices.SortFunc(out, func(a, b namedPattern) int { return strings.Compare(a.name, b.name) })
	return out
}

func SensitiveFieldName(name string) bool {
	lower := strings.ToLower(name)
	for _, pattern := range fieldPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}
	return false
}

func LooksSensitive(value string) bool {
	if opaqueToken.MatchString(value) && !recordID.MatchString(value) {
		return true
	}
	for _, pattern := range valuePatterns {
		if pattern.regex.MatchString(value) {
			return true
		}
	}
	return false
}

func Mask(value string) string {
	if len(value) > 4 {
		return value[:1] + strings.Repeat("*", len(value)-2) + value[len(value)-1:]
	}
	return stars
}

func MaskField(name, value string) (string, bool) {
	if value == "" {
		return value, false
	}
	if (name != "" && SensitiveFieldName(name)) || LooksSensitive(value) {
		return Mask(value), true
	}
	return value, false
}

func MaskText(text string) (string, bool) {
	if text == "" {
		return text, false
	}
	masked := false
	for _, pattern := range valuePatterns {
		text = pattern.regex.ReplaceAllStringFunc(text, func(match string) string {
			masked = true
			return Mask(match)
		})
	}
	return text, masked
}

type Scanner struct{}

func (Scanner) MaskField(name, value string) (string, bool) {
	return MaskField(name, value)
}

func (Scanner) MaskText(text string) (string, bool) {
	return MaskText(text)
}
