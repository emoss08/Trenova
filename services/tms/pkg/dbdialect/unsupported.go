package dbdialect

import (
	"fmt"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func Unsupported(kind Kind, capability Capability) *errortypes.NotImplementedError {
	return errortypes.NewNotImplementedError(
		fmt.Sprintf(
			"%s is not available when running on %s; configure database.driver=postgres to use this feature",
			capability.DisplayName(),
			kind.DisplayName(),
		),
	).WithDriver(kind.String()).WithCapability(string(capability))
}

func Require(kind Kind, capability Capability) error {
	if kind.Supports(capability) {
		return nil
	}

	return Unsupported(kind, capability)
}
