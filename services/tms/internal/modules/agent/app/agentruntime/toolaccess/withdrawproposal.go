package toolaccess

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetools"
)

func (s *Access) declareWithdrawal(set *ToolSet, delegated bool) {
	if delegated || !slices.ContainsFunc(set.Allowed, s.filesProposals) {
		return
	}

	set.add(runtimetools.WithdrawProposalSpec())
}

func (s *Access) filesProposals(name string) bool {
	_, ok := s.actionTools.Get(name)

	return ok
}
