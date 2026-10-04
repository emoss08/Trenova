package emailutils

import (
	"errors"
	"net/mail"
	"strings"
	"unicode/utf8"
)

const (
	MaxAddressLength   = 254
	MaxLocalPartLength = 64
	MaxDomainLength    = 253
	maxLabelLength     = 63
)

var (
	ErrEmpty            = errors.New("email address is empty")
	ErrTooLong          = errors.New("email address is too long")
	ErrInvalidSyntax    = errors.New("email address is not valid")
	ErrInvalidLocalPart = errors.New("email address local part is not valid")
	ErrInvalidDomain    = errors.New("email address domain is not valid")
)

var googleMailDomains = map[string]struct{}{
	"gmail.com":      {},
	"googlemail.com": {},
}

type Address struct {
	Original   string
	Lower      string
	LocalPart  string
	Domain     string
	Normalized string
}

func Parse(raw string) (Address, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Address{}, ErrEmpty
	}
	if len(trimmed) > MaxAddressLength {
		return Address{}, ErrTooLong
	}
	if !utf8.ValidString(trimmed) || strings.ContainsAny(trimmed, " \t\r\n<>\"(),;:\\[]") {
		return Address{}, ErrInvalidSyntax
	}

	parsed, err := mail.ParseAddress(trimmed)
	if err != nil || parsed.Address != trimmed || parsed.Name != "" {
		return Address{}, ErrInvalidSyntax
	}

	at := strings.LastIndexByte(trimmed, '@')
	if at <= 0 || at == len(trimmed)-1 {
		return Address{}, ErrInvalidSyntax
	}

	lower := strings.ToLower(trimmed)
	local := lower[:at]
	domain := lower[at+1:]

	if err = validateLocalPart(local); err != nil {
		return Address{}, err
	}
	if err = ValidateDomain(domain); err != nil {
		return Address{}, err
	}

	normalized := normalizedAddress(local, domain)
	if normalized == "" {
		return Address{}, ErrInvalidLocalPart
	}

	return Address{
		Original:   trimmed,
		Lower:      lower,
		LocalPart:  local,
		Domain:     domain,
		Normalized: normalized,
	}, nil
}

func Normalize(raw string) (string, error) {
	addr, err := Parse(raw)
	if err != nil {
		return "", err
	}

	return addr.Normalized, nil
}

func Validate(raw string) error {
	_, err := Parse(raw)
	return err
}

func Domain(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	at := strings.LastIndexByte(trimmed, '@')
	if at < 0 || at == len(trimmed)-1 {
		return ""
	}

	return trimmed[at+1:]
}

func LocalPart(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	at := strings.LastIndexByte(trimmed, '@')
	if at <= 0 {
		return ""
	}

	return trimmed[:at]
}

func DomainAllowed(domain string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}

	domain = strings.ToLower(strings.TrimSpace(domain))
	for _, candidate := range allowed {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if domain == candidate || strings.HasSuffix(domain, "."+candidate) {
			return true
		}
	}

	return false
}

func ValidateDomain(domain string) error {
	if domain == "" || len(domain) > MaxDomainLength {
		return ErrInvalidDomain
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return ErrInvalidDomain
	}

	for _, label := range labels {
		if !validLabel(label) {
			return ErrInvalidDomain
		}
	}

	if !validTopLevelLabel(labels[len(labels)-1]) {
		return ErrInvalidDomain
	}

	return nil
}

func validateLocalPart(local string) error {
	if local == "" || len(local) > MaxLocalPartLength {
		return ErrInvalidLocalPart
	}
	if local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return ErrInvalidLocalPart
	}

	for i := range len(local) {
		if !isAtextByte(local[i]) && local[i] != '.' {
			return ErrInvalidLocalPart
		}
	}

	return nil
}

func isAtextByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}

	return strings.IndexByte("!#$%&'*+/=?^_`{|}~-", c) >= 0
}

func validLabel(label string) bool {
	if label == "" || len(label) > maxLabelLength {
		return false
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}

	for i := range len(label) {
		c := label[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}

	return true
}

func validTopLevelLabel(label string) bool {
	if len(label) < 2 {
		return false
	}
	if strings.HasPrefix(label, "xn--") {
		return true
	}

	for i := range len(label) {
		if label[i] < 'a' || label[i] > 'z' {
			return false
		}
	}

	return true
}

func normalizedAddress(local, domain string) string {
	if _, ok := googleMailDomains[domain]; !ok {
		return local + "@" + domain
	}

	if plus := strings.IndexByte(local, '+'); plus >= 0 {
		local = local[:plus]
	}
	local = strings.ReplaceAll(local, ".", "")
	if local == "" {
		return ""
	}

	return local + "@gmail.com"
}
