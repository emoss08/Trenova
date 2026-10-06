package passwordutils

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/emailutils"
)

const (
	MinLength = 12
	MaxBytes  = 72
)

var (
	ErrTooShort      = errors.New("password must be at least 12 characters")
	ErrTooLong       = errors.New("password must be 72 bytes or fewer")
	ErrBlank         = errors.New("password cannot be only spaces")
	ErrContainsEmail = errors.New("password must not contain the email address")
	ErrCommon        = errors.New("password is too common")
)

func CheckPolicy(password string, address *emailutils.Address) error {
	switch {
	case utf8.RuneCountInString(password) < MinLength:
		return ErrTooShort
	case len(password) > MaxBytes:
		return ErrTooLong
	case strings.TrimSpace(password) == "":
		return ErrBlank
	}

	if address != nil && address.Lower != "" {
		lowered := strings.ToLower(strings.TrimSpace(password))
		if lowered == address.Lower || ContainsIgnoringCase(password, address.LocalPart) {
			return ErrContainsEmail
		}
	}

	if IsCommon(password) {
		return ErrCommon
	}

	return nil
}
