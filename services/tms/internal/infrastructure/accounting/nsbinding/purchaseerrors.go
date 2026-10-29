package nsbinding

import (
	"errors"
)

var ErrPurchaseDocumentType = errors.New(
	"netsuite: the purchase document type is neither a vendor bill nor a vendor credit",
)
