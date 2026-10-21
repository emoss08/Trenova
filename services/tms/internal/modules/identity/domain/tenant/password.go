package tenant

import (
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/passwordutils"
)

func ValidateNewPassword(
	multiErr *errortypes.MultiError,
	field string,
	password string,
	emailAddress string,
) {
	address, _ := emailutils.Parse(emailAddress)
	err := passwordutils.CheckPolicy(password, &address)
	switch {
	case err == nil:
	case errors.Is(err, passwordutils.ErrTooShort):
		multiErr.Add(field, errortypes.ErrInvalid, "Password must be at least 12 characters")
	case errors.Is(err, passwordutils.ErrTooLong):
		multiErr.Add(field, errortypes.ErrInvalid, "Password must be 72 bytes or fewer")
	case errors.Is(err, passwordutils.ErrBlank):
		multiErr.Add(field, errortypes.ErrInvalid, "Password cannot be only spaces")
	case errors.Is(err, passwordutils.ErrContainsEmail):
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Password must not contain the account's email address",
		)
	case errors.Is(err, passwordutils.ErrCommon):
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"This password is too common. Choose something harder to guess",
		)
	default:
		multiErr.Add(field, errortypes.ErrInvalid, "Password does not meet the password policy")
	}
}

func CheckNewPassword(field, password, emailAddress string) error {
	multiErr := errortypes.NewMultiError()
	ValidateNewPassword(multiErr, field, password, emailAddress)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
