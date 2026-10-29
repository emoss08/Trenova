package urlutils

import (
	"net/url"
	"regexp"
	"strings"
)

func ParseAbsoluteHTTP(raw string) (*url.URL, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return nil, false
	}

	switch parsed.Scheme {
	case "http", "https":
		return parsed, true
	default:
		return nil, false
	}
}

func HostMatches(host string, patterns []string) bool {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if normalized == "" {
		return false
	}
	for _, pattern := range patterns {
		candidate := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(pattern)), ".")
		if candidate == "" {
			continue
		}
		if suffix, wildcard := strings.CutPrefix(candidate, "*."); wildcard {
			if suffix != "" && strings.HasSuffix(normalized, "."+suffix) {
				return true
			}
			continue
		}
		if normalized == candidate {
			return true
		}
	}
	return false
}

var hostPattern = regexp.MustCompile(
	`^(\*\.)?([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`,
)

func IsHostPattern(domain string) bool {
	return hostPattern.MatchString(domain)
}
