package consolidationgen

import "github.com/rotisserie/eris"

var (
	ErrInvalidFormat              = eris.New("invalid consolidation format")
	ErrInvalidConsolidationNumber = eris.New("invalid consolidation number")
)
