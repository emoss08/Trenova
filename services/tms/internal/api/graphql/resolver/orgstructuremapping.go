package resolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/orgstructureservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// toTeamMembers renders a manager's team. The name is joined here rather than
// in SQL so it reads the same way it does everywhere else a worker is named.
func toTeamMembers(rows []repositories.TeamMemberRow) []*gqlmodel.TeamMember {
	members := make([]*gqlmodel.TeamMember, 0, len(rows))
	for _, row := range rows {
		members = append(members, &gqlmodel.TeamMember{
			WorkerID:         row.WorkerID.String(),
			Name:             row.FirstName + " " + row.LastName,
			Status:           row.Status,
			FleetCodeID:      idPtr(row.FleetCodeID),
			FleetCode:        row.FleetCode,
			FleetColor:       row.FleetColor,
			PositionID:       idPtr(row.PositionID),
			PositionTitle:    row.PositionTitle,
			Direct:           row.Direct,
			ManagerID:        idPtr(row.ManagerID),
			ComplianceStatus: row.ComplianceStatus,
			TrainingHealth:   row.TrainingHealth,
			SafetyRating:     row.SafetyRating,
			HireDate:         int(row.HireDate),
			TerminationDate:  intPtr(row.TerminationDate),
		})
	}
	return members
}

func toPositionHolders(rows []repositories.PositionHolderRow) []*gqlmodel.PositionHolder {
	out := make([]*gqlmodel.PositionHolder, 0, len(rows))
	for _, row := range rows {
		out = append(out, &gqlmodel.PositionHolder{
			Kind:   gqlmodel.PositionHolderKind(row.Kind),
			ID:     row.ID.String(),
			Name:   row.Name,
			Status: row.Status,
			Detail: row.Detail,
		})
	}
	return out
}

// assignRequest is the shared front of the two assignment mutations: the
// update grant on positions, the holder's id, and the position or its absence.
func (r *Resolver) assignRequest(
	ctx context.Context,
	holderField string,
	holderID string,
	positionID *string,
) (*orgstructureservice.AssignRequest, error) {
	authCtx, err := r.requirePermission(ctx, permission.ResourceJobPosition, permission.OpUpdate)
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
	position, err := optionalID(positionID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"positionId",
			errortypes.ErrInvalid,
			"Position is invalid",
		)
	}

	return &orgstructureservice.AssignRequest{
		TenantInfo: tenantInfo(authCtx),
		HolderID:   holder,
		PositionID: position,
		UserID:     authCtx.UserID,
	}, nil
}
