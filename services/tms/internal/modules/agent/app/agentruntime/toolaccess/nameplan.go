package toolaccess

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
)

func (s *Access) declareNamePlan(set *ToolSet, delegated bool) {
	if delegated || !slices.ContainsFunc(set.Allowed, s.filesProposals) {
		return
	}

	set.add(runtimetools.NamePlanSpec())
}
