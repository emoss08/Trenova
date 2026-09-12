package dbdialect

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

func Unsupported(kind Kind, capability Capability) *errortypes.NotImplementedError {
	return errortypes.NewNotImplementedError(
		"{0} is not available when running on {1}; configure database.driver=postgres to use this feature",
		capability.DisplayName(),
		kind.DisplayName(),
	).WithDriver(kind.String()).WithCapability(string(capability))
}

func Require(kind Kind, capability Capability) error {
	if kind.Supports(capability) {
		return nil
	}

	return Unsupported(kind, capability)
}
