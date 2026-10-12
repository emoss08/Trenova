package shipmentservice

import "github.com/emoss08/trenova/pkg/errortypes"

var errQuickFiltersUnavailable = errortypes.NewBusinessError(
	"Quick filters are not available in this process",
)
