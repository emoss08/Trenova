package slugutil

import (
	"regexp"
	"strconv"
	"strings"
)

const LoginSlugMaxLength = 100

var nonSlugCharsRE = regexp.MustCompile(`[^a-z0-9]+`)

func NormalizeLoginSlug(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = nonSlugCharsRE.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		normalized = "organization"
	}
	if len(normalized) > LoginSlugMaxLength {
		normalized = strings.Trim(normalized[:LoginSlugMaxLength], "-")
	}
	if normalized == "" {
		return "organization"
	}
	return normalized
}

func CandidateLoginSlug(base string, sequence int) string {
	base = NormalizeLoginSlug(base)
	if sequence <= 1 {
		return base
	}

	suffix := "-" + strconv.Itoa(sequence)
	maxBaseLength := LoginSlugMaxLength - len(suffix)
	if maxBaseLength < 1 {
		maxBaseLength = 1
	}

	if len(base) > maxBaseLength {
		base = strings.Trim(base[:maxBaseLength], "-")
	}
	if base == "" {
		base = "organization"
		if len(base) > maxBaseLength {
			base = base[:maxBaseLength]
		}
	}

	return base + suffix
}
