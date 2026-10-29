package bcsession

import (
	"errors"
)

var (
	ErrNoAccess = errors.New(
		"the signed-in user must be able to open Business Central to connect it",
	)
)
