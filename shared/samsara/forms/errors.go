package forms

import "errors"

var (
	ErrSubmissionIDRequired    = errors.New("form submission id is required")
	ErrSubmissionIDsRequired   = errors.New("form submission ids are required")
	ErrSubmissionIDsTooMany    = errors.New("form submission ids must contain at most 100 entries")
	ErrTemplateIDsTooMany      = errors.New("form template ids must contain at most 100 entries")
	ErrStreamStartTimeRequired = errors.New("form submissions stream startTime is required")
	ErrStreamTimeRangeInvalid  = errors.New(
		"form submissions stream endTime must not be before startTime",
	)
	ErrStreamFilterTooMany = errors.New(
		"form submissions stream filters must contain at most 50 ids",
	)
	ErrIncludeInvalid = errors.New("form submissions include value is invalid")
)
