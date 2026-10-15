package edi

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	redactedMarker     = "[redacted]"
	redactedPEMMarker  = "[redacted key or certificate]"
	isaSecurityPadding = 10
)

var (
	pemBlockRe  = regexp.MustCompile(`(?s)-{5}BEGIN [A-Z0-9 ]+-{5}.*?-{5}END [A-Z0-9 ]+-{5}`)
	secretLabel = regexp.MustCompile(
		`(?i)\b(password|passphrase|pass\s*word|pwd|passcode|secret|private\s+key|api\s+key|access\s+key|token|shared\s+key)(s?)\b(\s*(?:is|:|=|-)?\s*)([^\s,;]+)`,
	)
)

func RedactSecrets(value string) string {
	value = pemBlockRe.ReplaceAllString(value, redactedPEMMarker)
	value = redactInterchangeSecurity(value)
	return secretLabel.ReplaceAllString(value, "$1$2$3"+redactedMarker)
}

func redactInterchangeSecurity(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for {
		index := strings.Index(value, "ISA")
		if index < 0 || index+3 >= len(value) {
			builder.WriteString(value)
			return builder.String()
		}
		builder.WriteString(value[:index])
		separator := value[index+3]
		if unicode.IsLetter(rune(separator)) || unicode.IsDigit(rune(separator)) ||
			unicode.IsSpace(rune(separator)) {
			builder.WriteString(value[index : index+3])
			value = value[index+3:]
			continue
		}
		parts := strings.SplitN(value[index:], string(separator), 6)
		if len(parts) < 6 {
			builder.WriteString(value[index:])
			return builder.String()
		}
		parts[2] = maskElement(parts[1], parts[2])
		parts[4] = maskElement(parts[3], parts[4])
		builder.WriteString(strings.Join(parts[:5], string(separator)))
		builder.WriteByte(separator)
		value = parts[5]
	}
}

func maskElement(qualifier, element string) string {
	if strings.TrimSpace(qualifier) == "00" || strings.TrimSpace(element) == "" {
		return element
	}
	return strings.Repeat("X", max(len(element), isaSecurityPadding))
}
