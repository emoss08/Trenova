package assistanttest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
)

type StubDecisionRepo struct {
	agentports.AgentDecisionRepository

	Decisions []*agentdecision.AgentDecision
	Asked     [][]pulid.ID
}

func (s *StubDecisionRepo) ListByProposals(
	_ context.Context,
	req agentports.ListAgentDecisionsByProposalsRequest,
) ([]*agentdecision.AgentDecision, error) {
	s.Asked = append(s.Asked, req.ProposalIDs)

	return s.Decisions, nil
}

func DecisionWithNote(
	proposalID pulid.ID,
	decision agentdecision.DecisionType,
	note string,
) *agentdecision.AgentDecision {
	id := proposalID

	return &agentdecision.AgentDecision{
		ID:         pulid.MustNew("ad_"),
		ProposalID: &id,
		Decision:   decision,
		ReasonCode: "rejected_in_conversation",
		Note:       note,
	}
}
