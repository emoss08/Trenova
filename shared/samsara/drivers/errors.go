package drivers

import "errors"

var (
	ErrDriverIDRequired              = errors.New("driver id is required")
	ErrDriverNameRequired            = errors.New("driver name is required")
	ErrDriverNameTooLong             = errors.New("driver name must be at most 255 characters")
	ErrDriverUsernameRequired        = errors.New("driver username is required")
	ErrDriverUsernameTooLong         = errors.New("driver username must be at most 189 characters")
	ErrDriverUsernameInvalid         = errors.New("driver username may not contain spaces or '@'")
	ErrDriverPasswordRequired        = errors.New("driver password is required")
	ErrListLimitInvalid              = errors.New("drivers limit must be between 1 and 512")
	ErrDriverActivationStatusInvalid = errors.New("driver activation status is invalid")
)
