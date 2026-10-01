package base

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// assignRequest is the shared front of the two assignment mutations: the
// update grant on positions, the holder's id, and the position or its absence.
func (r *Resolver) AssignRequest(
	ctx context.Context,
	holderField string,
	holderID string,
	positionID *string,
) (*orgstructureservice.AssignRequest, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceJobPosition, permission.OpUpdate)
	if err != nil {
		return nil, err
	}

	holder, err := pulid.MustParse(holderID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			holderField,
			errortypes.ErrInvalid,
			"Somebody has to be named",
		)
	}
	position, err := OptionalID(positionID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"positionId",
			errortypes.ErrInvalid,
			"Position is invalid",
		)
	}

	return &orgstructureservice.AssignRequest{
		TenantInfo: TenantInfo(authCtx),
		HolderID:   holder,
		PositionID: position,
		UserID:     authCtx.UserID,
	}, nil
}
