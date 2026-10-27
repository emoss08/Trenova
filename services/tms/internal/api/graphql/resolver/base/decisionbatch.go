package base

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func BatchDecisionRequest(
	ids []string,
	input *gqlmodel.DecideAgentProposalsInput,
	tenant pagination.TenantInfo,
) (*agentports.DecideAgentProposalsRequest, error) {
	proposalIDs := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		parsed, err := pulid.MustParse(id)
		if err != nil {
			return nil, err
		}
		proposalIDs = append(proposalIDs, parsed)
	}

	digests, err := previewDigestsByProposal(input.PreviewDigests)
	if err != nil {
		return nil, err
	}

	reason := ""
	if input.ReasonCode != nil {
		reason = *input.ReasonCode
	}

	return &agentports.DecideAgentProposalsRequest{
		ProposalIDs:    proposalIDs,
		Decision:       input.Decision,
		ReasonCode:     reason,
		Note:           stringutils.FromPtr(input.Note),
		TenantInfo:     tenant,
		PreviewDigests: digests,
	}, nil
}

func BatchDecisionResults(
	results []agentports.AgentProposalDecisionResult,
) []*gqlmodel.AgentProposalDecisionResult {
	out := make([]*gqlmodel.AgentProposalDecisionResult, 0, len(results))
	for i := range results {
		out = append(out, DecisionResult(&results[i]))
	}

	return out
}

func DecisionResult(
	result *agentports.AgentProposalDecisionResult,
) *gqlmodel.AgentProposalDecisionResult {
	out := &gqlmodel.AgentProposalDecisionResult{
		ProposalID: result.ProposalID.String(),
		Decision:   result.Decision,
		Executed:   result.Executed,
		Refused:    result.Refused,
	}
	if result.Error != "" {
		message := result.Error
		out.Error = &message
	}

	return out
}

func PlanDecisionResult(plan *agent.AgentPlan) *gqlmodel.AgentPlanDecisionResult {
	return &gqlmodel.AgentPlanDecisionResult{
		Plan:    plan,
		Refused: agent.PlanTargetChanged(plan),
	}
}

func previewDigestsByProposal(
	inputs []*gqlmodel.AgentProposalPreviewDigestInput,
) (map[pulid.ID]string, error) {
	digests := make(map[pulid.ID]string, len(inputs))
	for i, input := range inputs {
		if input == nil {
			continue
		}
		id, err := pulid.Parse(input.ProposalID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("previewDigests[%d].proposalId", i),
				errortypes.ErrInvalid,
				"Proposal identifier is invalid",
			)
		}
		if _, dup := digests[id]; dup {
			return nil, errortypes.NewValidationError(
				fmt.Sprintf("previewDigests[%d].proposalId", i),
				errortypes.ErrInvalid,
				"A proposal's digest is listed twice",
			)
		}
		digests[id] = input.Digest
	}

	return digests, nil
}
