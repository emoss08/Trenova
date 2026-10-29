package bcsession

import (
	"errors"
)

var ErrPurchaseDocumentType = errors.New(
	"businesscentral: the purchase document type is neither a bill nor a vendor credit",
)
