package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func InvalidIDError(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

func EmptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// rotaManagerIDs is who the caller answers for. A team-scoped board that could
// not resolve the reporting line would widen to the whole roster, so the
// absence is refused rather than ignored.
func (r *Resolver) RotaManagerIDs(
	ctx context.Context,
	userID pulid.ID,
	tenant pagination.TenantInfo,
) ([]pulid.ID, error) {
	if r.OrgStructureService == nil {
		return nil, errortypes.NewAuthorizationError(
			"Your access is limited to your own team, which cannot be checked right now",
		)
	}

	ids, _, err := r.OrgStructureService.ManagerIDs(
		ctx,
		tenant,
		userID,
		worker.ApprovalScopeAll,
		0,
	)
	if err != nil {
		return nil, err
	}

	return ids, nil
}
