package compliance

import "errors"

var (
	ErrListLimitInvalid = errors.New("compliance limit must be between 1 and 512")
)
