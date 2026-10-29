package shipmentlifecycle

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

func cancelRequestRequired() error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add("request", errortypes.ErrRequired, "Cancel request is required")

	return multiErr
}
