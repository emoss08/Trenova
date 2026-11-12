package proposalpreviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/sourcegraph/conc/pool"
)

// batchConcurrency is how many of a batch's previews draft at once. Each
// holds a read-only snapshot for up to ReadTimeout, so the bound keeps one
// person's batch to a few connections.
const batchConcurrency = 4

// ForProposals previews several proposals for one reader, in the order
// given, for deciding them together. Pending proposals draft a few at a time
// and are compared with their baselines in one read; a decided proposal
// returns the preview it was decided on. A proposal that cannot be
// previewed carries its error and the rest are previewed regardless.
func (s *Service) ForProposals(
	ctx context.Context,
	req *agentruntimeports.ProposalPreviewsRequest,
) []agentruntimeports.ProposalPreviewOutcome {
	if req == nil || len(req.Proposals) == 0 {
		return []agentruntimeports.ProposalPreviewOutcome{}
	}

	actor := viewerActor(req.Viewer)
	reader := s.accessFor(req.Viewer)
	outcomes := make([]agentruntimeports.ProposalPreviewOutcome, len(req.Proposals))
	drafts := make([]*draft, len(req.Proposals))

	work := pool.New().WithMaxGoroutines(batchConcurrency)
	for i, proposal := range req.Proposals {
		outcomes[i].ProposalID = proposal.ID
		if err := assertTenant(
			proposal.OrganizationID,
			proposal.BusinessUnitID,
			actor,
		); err != nil {
			outcomes[i].Err = err
			continue
		}
		if proposal.Status != agent.ProposalStatusPending {
			work.Go(func() {
				outcomes[i].Preview, outcomes[i].Err = s.recordedFor(ctx, proposal, reader)
			})
			continue
		}
		work.Go(func() {
			drafts[i], outcomes[i].Err = s.draftStep(ctx, proposal, actor)
		})
	}
	work.Wait()

	pending := make([]*draft, 0, len(drafts))
	for _, d := range drafts {
		if d != nil {
			pending = append(pending, d)
		}
	}
	if len(pending) > 0 {
		s.markChangedSinceProposed(ctx, pagination.TenantInfo{
			OrgID: actor.OrganizationID,
			BuID:  actor.BusinessUnitID,
		}, pending)
	}

	for i, d := range drafts {
		if d == nil {
			continue
		}
		outcomes[i].Preview, outcomes[i].Err = s.finish(ctx, d, reader)
	}
	for i := range outcomes {
		if outcomes[i].Preview != nil {
			outcomes[i].Preview.ReviewRequired = s.reviewRequired(req.Proposals[i], nil)
		}
	}

	return outcomes
}
