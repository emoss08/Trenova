package agentextension

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MinSearchQueryLength = 3
	MaxSearchQueryLength = 400
	MaxPageURLLength     = 2048
	minPhoneDigits       = 10
)

var (
	ErrQueryTooShort    = errors.New("search query is too short; describe what you are looking for")
	ErrQueryTooLong     = errors.New("search query is longer than 400 characters; keep it to the question itself")
	ErrQueryHasRecordID = errors.New(
		"search queries leave Trenova, so they cannot include Trenova record IDs; describe the topic in general terms instead",
	)
	ErrQueryHasEmail = errors.New(
		"search queries leave Trenova, so they cannot include email addresses; describe the topic in general terms instead",
	)
	ErrQueryHasPhone = errors.New(
		"search queries leave Trenova, so they cannot include phone numbers; describe the topic in general terms instead",
	)
	ErrPageURLInvalid = errors.New("the page address is not a valid http or https URL")
)

var (
	recordIDPattern = regexp.MustCompile(`(?i)\b[a-z]{2,8}_[0-9a-hjkmnp-tv-z]{26}\b`)
	emailPattern    = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phonePattern    = regexp.MustCompile(`\+?\(?\d[\d\s().\-]{8,}\d`)
	officialSuffix  = []string{".gov", ".mil", ".gc.ca", ".gob.mx"}
)

func CheckSearchQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	length := utf8.RuneCountInString(trimmed)
	switch {
	case length < MinSearchQueryLength:
		return ErrQueryTooShort
	case length > MaxSearchQueryLength:
		return ErrQueryTooLong
	case recordIDPattern.MatchString(trimmed):
		return ErrQueryHasRecordID
	case emailPattern.MatchString(trimmed):
		return ErrQueryHasEmail
	}

	for _, candidate := range phonePattern.FindAllString(trimmed, -1) {
		if len(stringutils.DigitsOnly(candidate)) >= minPhoneDigits && !looksLikeDateRange(candidate) {
			return ErrQueryHasPhone
		}
	}

	return nil
}

func looksLikeDateRange(value string) bool {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ' ' || r == '-' || r == '.' || r == '(' || r == ')'
	})
	if len(fields) < 2 {
		return false
	}
	for _, field := range fields {
		if len(field) != 4 {
			return false
		}
	}

	return true
}

func ParsePageURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > MaxPageURLLength {
		return nil, ErrPageURLInvalid
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return nil, ErrPageURLInvalid
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return nil, ErrPageURLInvalid
	}

	return parsed, nil
}

func SiteOf(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}

	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
}

func IsOfficialSite(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	if host == "" {
		return false
	}
	for _, suffix := range officialSuffix {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}

	return false
}
