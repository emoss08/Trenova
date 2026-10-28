package accountinginbound

import (
	"errors"
)

var (
	ErrInboundChangeClosed = errors.New(
		"the change was already applied, ignored or superseded",
	)
	ErrInboundChangeNotProposed = errors.New("only a proposed change can be applied")
	ErrInboundNoteRequired      = errors.New("a note is required to ignore a change")
	ErrInboundNotApplicable     = errors.New(
		"this change cannot be applied until what it pays matches Trenova",
	)
)
