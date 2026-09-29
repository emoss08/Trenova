package urlutils

import (
	"net/url"
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
