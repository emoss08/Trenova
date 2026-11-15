package supportaccess

import (
	"errors"
	"strings"
)

const (
	maxEmailLength = 254
	maxLocalLength = 64
)

var (
	ErrEmailInvalid     = errors.New("not a plain ASCII email address")
	ErrEmailNotStaff    = errors.New("not at the Trenova staff domain")
	ErrStaffDomainUnset = errors.New("the staff domain is not configured")
)

// NormalizeStaffEmail is the one way a staff email is compared, on both sides of support
// access: the address an administrator grants and the address the control plane asserts. It
// refuses anything but plain ASCII (so no Unicode look-alike can equal a staff address after
// some later folding), lower-cases ASCII letters only, and requires the domain to be exactly
// the configured staff domain, never a subdomain of it.
func NormalizeStaffEmail(raw, staffDomain string) (string, error) {
	domain, err := normalizeDomain(staffDomain)
	if err != nil {
		return "", err
	}

	email := strings.TrimSpace(raw)
	if email == "" || len(email) > maxEmailLength {
		return "", ErrEmailInvalid
	}
	for i := 0; i < len(email); i++ {
		if email[i] >= 0x80 || email[i] <= 0x20 || email[i] == 0x7f {
			return "", ErrEmailInvalid
		}
	}
	email = lowerASCII(email)

	local, host, ok := strings.Cut(email, "@")
	if !ok || strings.Contains(host, "@") {
		return "", ErrEmailInvalid
	}
	if !validLocal(local) {
		return "", ErrEmailInvalid
	}
	if host != domain {
		return "", ErrEmailNotStaff
	}

	return local + "@" + host, nil
}

func normalizeDomain(raw string) (string, error) {
	domain := lowerASCII(strings.TrimSpace(raw))
	if domain == "" {
		return "", ErrStaffDomainUnset
	}
	if !ValidDomain(domain) {
		return "", ErrStaffDomainUnset
	}
	return domain, nil
}

// ValidDomain reports whether d is a lower-case ASCII host name with at least two labels.
func ValidDomain(d string) bool {
	if len(d) == 0 || len(d) > 253 {
		return false
	}
	labels := strings.Split(d, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			c := label[i]
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

func validLocal(local string) bool {
	if local == "" || len(local) > maxLocalLength {
		return false
	}
	if local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return false
	}
	for i := 0; i < len(local); i++ {
		c := local[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		case c == '.', c == '_', c == '+', c == '-':
		default:
			return false
		}
	}
	return true
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
