package compliance

import "errors"

var (
	ErrListLimitInvalid              = errors.New("compliance limit must be between 1 and 512")
	ErrDateFormatInvalid             = errors.New("compliance date must be in YYYY-MM-DD format")
	ErrDateRangeInvalid              = errors.New("compliance endDate must not be before startDate")
	ErrDriverActivationStatusInvalid = errors.New("driver activation status is invalid")
	ErrExpandInvalid                 = errors.New("hos daily logs expand value is invalid")
	ErrTimeRangeRequired             = errors.New("startTime and endTime are required")
	ErrTimeRangeInvalid              = errors.New("endTime must not be before startTime")
)
