package decisionsresolver

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// previewDigestsByProposal reads a batch's digests, one per proposal.
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
