package factoringaction

import (
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const MaxFutureMovementSeconds = timeutils.SecondsPerDay

func RequireExecutor(actor *permissionports.RequestActor, action string) error {
	if err := RequireActor(actor, action); err != nil {
		return err
	}
	if actor.ExecutorUserID().IsNil() {
		return errortypes.NewAuthorizationError(
			"Factoring entries are booked to the ledger, so a person has to make or approve them",
		)
	}
	return nil
}

func SystemActor(tenantInfo pagination.TenantInfo) permissionports.RequestActor {
	return permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}
