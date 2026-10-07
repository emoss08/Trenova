package aicontrolresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/agentroster"
)

func rosterModel(stat *agentroster.Stat) *gqlmodel.AIAgentRosterStat {
	return &gqlmodel.AIAgentRosterStat{
		AgentID:        stat.AgentID.String(),
		RunsByDay:      stat.RunsByDay,
		Runs:           stat.Runs(),
		Approved:       stat.Approved,
		Modified:       stat.Modified,
		Rejected:       stat.Rejected,
		Failed:         stat.Failed,
		ShadowRecorded: stat.ShadowRecorded,
		ApprovalRate:   stat.ApprovalRate(),
	}
}
