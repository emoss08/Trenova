package cdcutils

import "errors"

var (
	ErrDeleteEventMissingBeforeData = errors.New(
		"delete event missing 'before' data for tenant extraction",
	)
	ErrEventMissingAfterData = errors.New("event missing 'after' data for tenant extraction")
)
