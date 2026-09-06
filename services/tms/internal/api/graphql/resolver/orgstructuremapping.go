package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
			ComplianceStatus: row.ComplianceStatus,
			TrainingHealth:   row.TrainingHealth,
			SafetyRating:     row.SafetyRating,
			HireDate:         int(row.HireDate),
			TerminationDate:  intPtr(row.TerminationDate),
		})
	}
	return members
}
