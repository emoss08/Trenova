package pronumbergen

import "github.com/rotisserie/eris"

var (
	ErrInvalidFormat    = eris.New("invalid pro number format")
	ErrInvalidProNumber = eris.New("invalid pro number")
)
