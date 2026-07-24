package compliance

import "errors"

var (
	ErrListLimitInvalid  = errors.New("compliance limit must be between 1 and 512")
	ErrDateFormatInvalid = errors.New("compliance date must be in YYYY-MM-DD format")
)
