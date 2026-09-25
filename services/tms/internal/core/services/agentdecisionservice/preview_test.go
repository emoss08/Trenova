package agentdecisionservice

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePreviews answers as the preview service would: stale when the pinned
// record has moved, and with the digest of what it shows.
type fakePreviews struct {
	services.ProposalPreviewService

	stale  bool
	digest string
	asked  []*services.ProposalPreviewRequest
}

func (f *fakePreviews) ForProposal(
	_ context.Context,
	req *services.ProposalPreviewRequest,
) (*agent.ProposalPreview, error) {
	f.asked = append(f.asked, req)
	version := int64(3)
	preview := &agent.ProposalPreview{
		Schema:        agent.PreviewSchemaVersion,
		ProposalID:    req.Proposal.ID,
		Tool:          req.Proposal.ToolName,
		Coverage:      agent.PreviewCoverageFull,
		TargetVersion: &version,
		Digest:        f.digest,
		Staleness:     &agent.PreviewStaleness{Pinned: true, ProposedVersion: 3, CurrentVersion: 3},
	}
	if f.stale {
		preview.Staleness.CurrentVersion = 5
	}

	return preview, nil
}

type statusRecorder struct {
	*traceProposals

	updated int
}

func (p *statusRecorder) UpdateStatus(
	ctx context.Context,
	req repositories.UpdateAgentProposalStatusRequest,
) (*agent.AgentProposal, error) {
	p.updated++

	return p.traceProposals.UpdateStatus(ctx, req)
}

func previewHarness(t *testing.T, previews *fakePreviews) (*decideHarness, *statusRecorder) {
	t.Helper()

	h := newDecideHarness(t)
	recorder := &statusRecorder{traceProposals: h.proposals}
	h.service.proposalRepo = recorder
	h.service.previews = previews

	return h, recorder
}

func decideWith(
	t *testing.T,
	h *decideHarness,
	decision agent.DecisionType,
	digest string,
) (*services.DecisionOutcome, error) {
	t.Helper()

	return h.service.DecideWithOutcome(t.Context(), &services.DecideAgentProposalRequest{
		ProposalID:    h.proposals.proposal.ID,
		Decision:      decision,
		ReasonCode:    "looks_right",
		PreviewDigest: digest,
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}, h.actor)
}

// An approval of a change whose record moved on used to be recorded as
// approved and then reported as "approved, but it did not run". It is now
// refused before anything is written.
func TestDecide_AStaleApprovalIsRefusedBeforeItIsRecorded(t *testing.T) {
	t.Parallel()

	h, recorder := previewHarness(t, &fakePreviews{stale: true, digest: strings.Repeat("a", 64)})

	_, err := decideWith(t, h, agent.DecisionAccepted, "")

	require.Error(t, err)
	var business *errortypes.BusinessError
	require.True(t, errors.As(err, &business), "a stale approval is a business refusal: %v", err)
	assert.Nil(t, h.decisions.created, "no decision is recorded")
	assert.Zero(t, recorder.updated, "the proposal stays pending")
	assert.Empty(t, h.proposals.executed, "nothing runs")
}

func TestDecide_ADigestThatNoLongerMatchesIsAConflictAndWritesNothing(t *testing.T) {
	t.Parallel()

	h, recorder := previewHarness(t, &fakePreviews{digest: strings.Repeat("a", 64)})

	_, err := decideWith(t, h, agent.DecisionAccepted, strings.Repeat("b", 64))

	require.Error(t, err)
	var conflict *errortypes.ConflictError
	require.True(t, errors.As(err, &conflict), "a changed preview is a conflict: %v", err)
	assert.Nil(t, h.decisions.created)
	assert.Zero(t, recorder.updated)
	assert.Empty(t, h.proposals.executed)
}

func TestDecide_RecordsWhatTheApproverSawAndWhetherTheyReviewedIt(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("c", 64)
	previews := &fakePreviews{digest: digest}
	h, _ := previewHarness(t, previews)

	outcome, err := decideWith(t, h, agent.DecisionAccepted, digest)
	require.NoError(t, err)
	require.NoError(t, outcome.ExecutionError)

	created := h.decisions.created
	require.NotNil(t, created)
	require.NotNil(t, created.Preview)
	assert.Equal(t, digest, created.PreviewDigest)
	assert.True(t, created.PreviewReviewed)
	require.NotNil(t, created.PreviewTargetVersion)
	assert.Equal(t, int64(3), *created.PreviewTargetVersion)
	require.Len(t, previews.asked, 1)
	assert.True(t, previews.asked[0].Deciding)
	assert.Equal(t, h.actor, previews.asked[0].Viewer.Actor, "previewed as the decider")
	require.NotNil(t, outcome.ExecutedTargetVersion)
	assert.Equal(t, int64(4), *outcome.ExecutedTargetVersion)
}

func TestDecide_AnApprovalWithoutADigestIsRecordedUnreviewed(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("d", 64)
	h, _ := previewHarness(t, &fakePreviews{digest: digest})

	_, err := decideWith(t, h, agent.DecisionAccepted, "")
	require.NoError(t, err)

	created := h.decisions.created
	require.NotNil(t, created)
	assert.Equal(t, digest, created.PreviewDigest, "the preview it ran against is recorded")
	assert.False(t, created.PreviewReviewed)
}

func TestDecide_ARejectionRecordsOnlyTheDigestItWasSent(t *testing.T) {
	t.Parallel()

	previews := &fakePreviews{stale: true, digest: strings.Repeat("e", 64)}
	h, _ := previewHarness(t, previews)

	sent := strings.Repeat("f", 64)
	_, err := decideWith(t, h, agent.DecisionRejected, sent)
	require.NoError(t, err, "a stale proposal can always be rejected")

	created := h.decisions.created
	require.NotNil(t, created)
	assert.Nil(t, created.Preview)
	assert.Equal(t, sent, created.PreviewDigest)
	assert.False(t, created.PreviewReviewed)
	assert.Empty(t, previews.asked)
}
