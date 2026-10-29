package settlementshared

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func RequireActor(actor *permissionports.RequestActor, operation string) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("{0} requires an authenticated user", operation)
	}
	return nil
}
