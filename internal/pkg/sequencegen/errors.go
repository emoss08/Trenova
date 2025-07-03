package sequencegen

import "github.com/rotisserie/eris"

// Errors
var (
	ErrSequenceUpdateConflict = eris.New("sequence update conflict")
	ErrInvalidYear            = eris.New("year out of range for int16")
	ErrInvalidMonth           = eris.New("month out of range for int16")
)
