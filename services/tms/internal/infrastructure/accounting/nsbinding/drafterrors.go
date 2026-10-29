package nsbinding

import (
	"errors"
)

var (
	ErrItemDraftRequired  = errors.New("netsuite: an item draft is required")
	ErrPartyDraftRequired = errors.New("netsuite: a customer or vendor draft is required")
	ErrNameRequired       = errors.New("netsuite: a name is required")
)
