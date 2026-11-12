package proposalrecorder

import (
	"context"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentproposalports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const standingReadLimit = 200

func (s *Service) standAside(
	ctx context.Context,
	actor *permissionports.RequestActor,
	group *runGroup,
	now int64,
) error {
	if s.standing == nil || group.definition == nil || !runsUnattended(group) {
		return nil
	}

	tools := make([]string, 0, len(group.proposals))
	for _, proposal := range group.proposals {
		if proposal.Status == agent.ProposalStatusPending &&
			!slices.Contains(tools, proposal.ToolName) {
			tools = append(tools, proposal.ToolName)
		}
	}
	if len(tools) == 0 {
		return nil
	}

	waiting, err := s.standing.ListStandingByDefinition(
		ctx,
		&agentproposalports.ListStandingByDefinitionRequest{
			TenantInfo:   actor.TenantInfo(),
			DefinitionID: group.definition.ID,
			ToolNames:    tools,
			Now:          now,
			Limit:        standingReadLimit,
		},
	)
	if err != nil {
		return err
	}

	asked := make(map[string]struct{}, len(waiting))
	for _, proposal := range waiting {
		if key, ok := sameWrite(proposal); ok {
			asked[key] = struct{}{}
		}
	}
	for _, proposal := range group.proposals {
		if proposal.Status != agent.ProposalStatusPending {
			continue
		}
		key, ok := sameWrite(proposal)
		if !ok {
			continue
		}
		if _, repeated := asked[key]; repeated {
			proposal.Status = agent.ProposalStatusSuperseded
		}
	}

	return nil
}

func sameWrite(proposal *agent.AgentProposal) (string, bool) {
	params, err := sonic.ConfigStd.MarshalToString(nonNilParams(proposal.ToolParams))
	if err != nil {
		return "", false
	}

	return proposal.ToolName + "\x00" + proposal.TargetResource + "\x00" +
		proposal.TargetID.String() + "\x00" + params, true
}
