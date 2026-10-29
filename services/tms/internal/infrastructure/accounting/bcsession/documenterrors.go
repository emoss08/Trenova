package bcsession

import (
	"errors"
)

const (
	ProviderName = "Business Central"
)

var (
	ErrDocumentKind = errors.New(
		"businesscentral: this record type is not a document Business Central holds",
	)
	ErrExternalID = errors.New(
		"businesscentral: changing a document needs the id Business Central gave it",
	)
	ErrLinesRequired = errors.New("businesscentral: a document needs at least one line")
)
