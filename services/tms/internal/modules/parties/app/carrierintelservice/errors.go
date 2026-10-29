package carrierintelservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func ErrorSummary(err error) string {
	var providerErr *carrierintelports.CarrierIntelProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Error()
	}
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return business.Error()
	}
	return "provider request failed"
}
