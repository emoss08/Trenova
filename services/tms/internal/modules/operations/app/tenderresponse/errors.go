package tenderresponse

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

// ErrOfferNoLongerAvailable is returned to any channel whose response arrived
// after the offer left its live window. The copy is deliberately identical for
// expired, superseded, withdrawn, and already-answered offers.
var ErrOfferNoLongerAvailable = errortypes.NewBusinessError(
	"This offer is no longer available",
)
