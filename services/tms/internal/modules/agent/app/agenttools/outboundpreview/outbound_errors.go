package outboundpreview

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
)

// ErrCustomerAlreadyTold is a skip rather than a failure: the customer
// has the update, which is the outcome the call wanted.
var ErrCustomerAlreadyTold = customerupdateservice.ErrAlreadyTold
