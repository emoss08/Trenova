package agentpreviewresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var proposalRead = permission.ResourceAgentProposal.String() + "|" + string(permission.OpRead)

func TestAgentProposalPreview_NeedsProposalRead(t *testing.T) {
	t.Parallel()

	denied := newPreviewHarness(t)
	_, err := denied.resolver.AgentProposalPreview(denied.ctx, denied.proposal.String(), nil)
	require.Error(t, err)
	assert.Zero(t, denied.previews.Proposals, "nothing is previewed for a reader without access")

	allowed := newPreviewHarness(t, proposalRead)
	mods := map[string]any{"message": "Call dispatch"}
	preview, err := allowed.resolver.AgentProposalPreview(allowed.ctx, allowed.proposal.String(), mods)
	require.NoError(t, err)
	assert.Equal(t, allowed.proposal, preview.ProposalID)
	assert.Equal(t, mods, allowed.previews.Mods)
	require.NotNil(t, allowed.previews.Viewer)
	assert.NotNil(t, allowed.previews.Viewer.Reads, "reads are checked for this reader")

	_, err = allowed.resolver.AgentPlanPreview(allowed.ctx, pulid.MustNew("apl_").String())
	require.NoError(t, err)
	_, err = denied.resolver.AgentPlanPreview(denied.ctx, pulid.MustNew("apl_").String())
	require.Error(t, err)
}

func TestMyProposalPreview_IsTheCallersOwnAlone(t *testing.T) {
	t.Parallel()

	withoutAssistant := newPreviewHarness(t, proposalRead)
	_, err := withoutAssistant.resolver.MyProposalPreview(
		withoutAssistant.ctx, withoutAssistant.proposal.String(), nil)
	require.Error(t, err, "the self-scoped query needs assistant:read")

	notYours := newPreviewHarness(t, assistantRead)
	_, err = notYours.resolver.MyProposalPreview(notYours.ctx, notYours.proposal.String(), nil)
	require.True(t, errortypes.IsNotFoundError(err), "someone else's proposal is not found")
	assert.Equal(t, []pulid.ID{notYours.proposal}, notYours.decisions.Asked)
	assert.Zero(t, notYours.previews.Proposals)

	yours := newPreviewHarness(t, assistantRead)
	yours.decisions.Yours = true
	_, err = yours.resolver.MyProposalPreview(yours.ctx, yours.proposal.String(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, yours.previews.Proposals)
}

func TestMyPlanPreview_IsTheCallersOwnAlone(t *testing.T) {
	t.Parallel()

	notYours := newPreviewHarness(t, assistantRead)
	_, err := notYours.resolver.MyPlanPreview(notYours.ctx, pulid.MustNew("apl_").String())
	require.True(t, errortypes.IsNotFoundError(err))
	assert.Zero(t, notYours.previews.Plans)

	yours := newPreviewHarness(t, assistantRead)
	yours.plans.Yours = true
	_, err = yours.resolver.MyPlanPreview(yours.ctx, pulid.MustNew("apl_").String())
	require.NoError(t, err)
	assert.Equal(t, 1, yours.previews.Plans)
}
