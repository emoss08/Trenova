package seqgen

import "errors"

var (
	ErrInvalidSequenceType     = errors.New("invalid sequence type")
	ErrSequenceFormatNil       = errors.New("sequence format cannot be nil")
	ErrSequenceCannotBeEmpty   = errors.New("sequence cannot be empty")
	ErrSequenceRequestRequired = errors.New("sequence request is required")
	ErrNoSequencesReturned     = errors.New("no sequences returned")
)
