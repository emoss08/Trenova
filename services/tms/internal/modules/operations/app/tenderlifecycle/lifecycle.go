package tenderlifecycle

import (
	"errors"
)

// errRollbackAccept aborts the acceptance transaction when the offer lost its
// CAS, unwinding the tender transition with it.
var errRollbackAccept = errors.New("tender accept rolled back: offer already terminal")
