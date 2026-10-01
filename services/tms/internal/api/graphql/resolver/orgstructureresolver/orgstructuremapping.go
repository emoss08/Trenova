package orgstructureresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

// toTeamMembers renders a manager's team. The name is joined here rather than
// in SQL so it reads the same way it does everywhere else a worker is named.
func toTeamMembers(rows []repositories.TeamMemberRow) []*gqlmodel.TeamMember {
	members := make([]*gqlmodel.TeamMember, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		members = append(members, &gqlmodel.TeamMember{
			WorkerID:         row.WorkerID.String(),
			Name:             row.FirstName + " " + row.LastName,
			Status:           row.Status,
			FleetCodeID:      base.IDPtr(row.FleetCodeID),
			FleetCode:        row.FleetCode,
			FleetColor:       row.FleetColor,
			PositionID:       base.IDPtr(row.PositionID),
			PositionTitle:    row.PositionTitle,
			Direct:           row.Direct,
			ManagerID:        base.IDPtr(row.ManagerID),
			ComplianceStatus: row.ComplianceStatus,
			TrainingHealth:   row.TrainingHealth,
			SafetyRating:     row.SafetyRating,
			HireDate:         int(row.HireDate),
			TerminationDate:  base.IntPtr(row.TerminationDate),
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
