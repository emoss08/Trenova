package forms

import "errors"

var (
	ErrSubmissionIDRequired    = errors.New("form submission id is required")
	ErrStreamStartTimeRequired = errors.New("form submissions stream startTime is required")
)
