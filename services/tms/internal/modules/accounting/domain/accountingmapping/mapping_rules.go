package accountingmapping

import (
	"errors"
)

var (
	ErrMappingNotProposed = errors.New("only a proposed mapping can be rejected")
	ErrMappingNotSet      = errors.New("the mapping has no accounting record to clear")
)

const (
	PrecheckConfidence  = 0.95
	ConfidentConfidence = 0.70
	MaxModelConfidence  = 0.90
	MaxCandidates       = 3
	maxReasonLength     = 500
)
