package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func batchDecisionRequest(
	ids []string,
	input *gqlmodel.DecideAgentProposalsInput,
	tenant pagination.TenantInfo,
) (*services.DecideAgentProposalsRequest, error) {
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

	return &services.DecideAgentProposalsRequest{
		ProposalIDs:    proposalIDs,
		Decision:       input.Decision,
		ReasonCode:     reason,
		TenantInfo:     tenant,
		PreviewDigests: digests,
	}, nil
}

func batchDecisionResults(
	results []services.AgentProposalDecisionResult,
) []*gqlmodel.AgentProposalDecisionResult {
	out := make([]*gqlmodel.AgentProposalDecisionResult, 0, len(results))
	for i := range results {
		result := &gqlmodel.AgentProposalDecisionResult{
			ProposalID: results[i].ProposalID.String(),
			Decision:   results[i].Decision,
			Executed:   results[i].Executed,
		}
		if results[i].Error != "" {
			message := results[i].Error
			result.Error = &message
		}
		out = append(out, result)
	}

	return out
}
