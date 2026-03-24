package vehicles

import "errors"

var (
	ErrListLimitInvalid = errors.New("vehicle stats limit must be between 1 and 512")
)
