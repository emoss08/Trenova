package seqgen

import (
	"errors"
)

var (
	ErrSequenceUpdateConflict  = errors.New("sequence update conflict")
	ErrInvalidSequenceType     = errors.New("invalid sequence type")
	ErrSequenceFormatNil       = errors.New("sequence format cannot be nil")
	ErrSequenceDoesNotMatch    = errors.New("sequence does not match expected prefix")
	ErrSequenceCannotBeEmpty   = errors.New("sequence cannot be empty")
	ErrInvalidCheckDigitFormat = errors.New("invalid check digit format")
	ErrSequenceTooShort        = errors.New("sequence too short for check digit validation")
	ErrUnexpectedEndOfSequence = errors.New("unexpected end of sequence")
	ErrMissingCheckDigit       = errors.New("missing check digit")
)
