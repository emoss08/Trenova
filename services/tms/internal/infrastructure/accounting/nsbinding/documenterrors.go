package nsbinding

import (
	"errors"
)

var (
	ErrDocumentKind = errors.New(
		"netsuite: this record type is not a document NetSuite holds",
	)
	ErrExternalID = errors.New(
		"netsuite: changing a document needs the id NetSuite gave it",
	)
	ErrRequestID     = errors.New("netsuite: a write needs the request id to key its external id")
	ErrLinesRequired = errors.New("netsuite: a document needs at least one line")
	ErrAmountInvalid = errors.New("netsuite: the amount must be positive")
)
