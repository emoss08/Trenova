package rptmetavalidator

import (
	"github.com/rotisserie/eris"
)

var (
	ErrEmptySQL        = eris.New("sql query cannot be empty")
	ErrNoPlaceholder   = eris.New("no placeholder found in sql query")
	ErrInvalidMetadata = eris.New("invalid metadata structure")
)
