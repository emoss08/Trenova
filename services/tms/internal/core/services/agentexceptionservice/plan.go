package agentexceptionservice

import (
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func PlanException(req *services.FlagAgentExceptionRequest) (*agent.AgentException, error) {
	entity := &agent.AgentException{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		RunID:           req.RunID,
		Category:        req.Category,
		Severity:        req.Severity,
		SubjectType:     req.SubjectType,
		SubjectID:       req.SubjectID,
		AttemptSummary:  req.AttemptSummary,
		Evidence:        req.Evidence,
		BlastRadius:     req.BlastRadius,
		ResolutionState: agent.ResolutionStateOpen,
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return entity, me
	}

	return entity, nil
}
