package proposalpreviewservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A batch previews each proposal in the order asked, each digested exactly
// as it would be on its own, so a batch approval checks the same digests a
// card would have sent.
func TestForProposals_PreviewsEachInTheOrderAskedWithItsOwnDigest(t *testing.T) {
	t.Parallel()

	f := newFixture()
	f.checker.review = true
	cancel := f.proposal(f.cancelParams())
	hold := f.proposal(map[string]any{"shipmentId": f.tool.state.ID.String(), "status": "Hold"})

	outcomes := f.svc.ForProposals(t.Context(), &agentruntimeports.ProposalPreviewsRequest{
		Proposals: []*agent.AgentProposal{hold, cancel},
		Viewer:    f.viewer(),
	})

	require.Len(t, outcomes, 2)
	assert.Equal(t, hold.ID, outcomes[0].ProposalID)
	assert.Equal(t, cancel.ID, outcomes[1].ProposalID)
	for _, outcome := range outcomes {
		require.NoError(t, outcome.Err)
		require.NotNil(t, outcome.Preview)
		assert.True(t, outcome.Preview.ReviewRequired, "each says whether its approval needs it")
	}
	assert.Equal(t, "Would set PRO PRO-100 to Hold.", outcomes[0].Preview.Summary)
	assert.Equal(t, "Would set PRO PRO-100 to Cancelled.", outcomes[1].Preview.Summary)

	for i, proposal := range []*agent.AgentProposal{hold, cancel} {
		alone, err := f.svc.ForProposal(t.Context(), &agentruntimeports.ProposalPreviewRequest{
			Proposal: proposal,
			Viewer:   f.viewer(),
		})
		require.NoError(t, err)
		assert.Equal(t, alone.Digest, outcomes[i].Preview.Digest,
			"a batch shows each change as its card would")
	}
}

// The baselines every pending proposal is compared with are read once for
// the batch, and every preview drafts in a read-only snapshot.
func TestForProposals_ReadsTheBaselinesOnceForTheBatch(t *testing.T) {
	t.Parallel()

	f := newFixture()
	proposals := make([]*agent.AgentProposal, 0, 6)
	for range 6 {
		proposals = append(proposals, f.proposal(f.cancelParams()))
	}

	outcomes := f.svc.ForProposals(t.Context(), &agentruntimeports.ProposalPreviewsRequest{
		Proposals: proposals,
		Viewer:    f.viewer(),
	})

	require.Len(t, outcomes, 6)
	assert.Equal(t, 1, f.baselines.listed)
	require.Len(t, f.db.opts, 6)
	for _, opts := range f.db.opts {
		assert.True(t, opts.ReadOnly, "a preview never writes")
	}
}

// A proposal that cannot be previewed, here one of another tenant's, carries
// its error, and the rest are previewed regardless; a decided proposal
// returns the preview it was decided on.
func TestForProposals_OneFailureLeavesTheOthersPreviewed(t *testing.T) {
	t.Parallel()

	f := newFixture()
	pending := f.proposal(f.cancelParams())
	foreign := f.proposal(f.cancelParams())
	foreign.OrganizationID = pulid.MustNew("org_")
	decided := f.proposal(f.cancelParams())
	decided.Status = agent.ProposalStatusExecuted
	f.decisions.decisions = []*agentdecision.AgentDecision{{
		ID:             pulid.MustNew("ad_"),
		OrganizationID: f.org,
		BusinessUnitID: f.bu,
		ProposalID:     &decided.ID,
		Preview: &agentpreview.ProposalPreview{
			Schema:     agentpreview.PreviewSchemaVersion,
			ProposalID: decided.ID,
			Summary:    "Would cancel PRO-100.",
			Coverage:   agentpreview.PreviewCoverageFull,
		},
		PreviewDigest: "d1",
	}}

	outcomes := f.svc.ForProposals(t.Context(), &agentruntimeports.ProposalPreviewsRequest{
		Proposals: []*agent.AgentProposal{pending, foreign, decided},
		Viewer:    f.viewer(),
	})

	require.Len(t, outcomes, 3)
	require.NoError(t, outcomes[0].Err)
	assert.Len(t, outcomes[0].Preview.Digest, 64)
	require.ErrorIs(t, outcomes[1].Err, ErrTenantMismatch)
	assert.Nil(t, outcomes[1].Preview)
	require.NoError(t, outcomes[2].Err)
	assert.True(t, outcomes[2].Preview.Recorded)
	assert.Equal(t, "d1", outcomes[2].Preview.Digest)
	assert.Len(t, f.tool.asked, 1, "only the pending proposal of this tenant is drafted")
}

func TestForProposals_AnEmptyBatchPreviewsNothing(t *testing.T) {
	t.Parallel()

	f := newFixture()

	assert.Empty(t, f.svc.ForProposals(t.Context(), &agentruntimeports.ProposalPreviewsRequest{
		Viewer: f.viewer(),
	}))
	assert.Zero(t, f.baselines.listed)
}
