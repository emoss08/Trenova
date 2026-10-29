package bcsession

import (
	"errors"
)

var (
	ErrItemDraftRequired  = errors.New("businesscentral: an item draft is required")
	ErrPartyDraftRequired = errors.New("businesscentral: a customer or vendor draft is required")
)
