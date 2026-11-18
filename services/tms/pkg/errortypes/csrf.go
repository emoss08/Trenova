package errortypes

import "errors"

type CSRFTokenInvalidError struct {
	BaseError
}

func NewCSRFTokenInvalidError() *CSRFTokenInvalidError {
	return &CSRFTokenInvalidError{
		BaseError: BaseError{
			Code:    ErrCSRFTokenInvalid,
			Message: "This page's security token is out of date. Reload the page to continue.",
		},
	}
}

func IsCSRFTokenInvalidError(err error) bool {
	_, ok := errors.AsType[*CSRFTokenInvalidError](err)
	return ok
}
