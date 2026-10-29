package agentdecisionservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gatedClasses() []agent.EgressClass {
	return []agent.EgressClass{
		agent.EgressMoney,
		agent.EgressCustomerVisible,
		agent.EgressDriverVisible,
		agent.EgressExternalRecipient,
		"",
	}
}

func TestDecide_AGatedApprovalWithoutADigestIsRefused(t *testing.T) {
	t.Parallel()

	for _, class := range gatedClasses() {
		t.Run(string(class), func(t *testing.T) {
			t.Parallel()

			h, recorder := previewHarness(t, &fakePreviews{digest: strings.Repeat("d", 64)})
			h.proposals.proposal.EgressClass = class

			_, err := decideWith(t, h, agentdecision.DecisionAccepted, "")

			require.Error(t, err)
			assert.True(
				t,
				agent.IsReviewRequired(err),
				"the refusal says to review it first: %v",
				err,
			)
			assert.Contains(t, err.Error(), "must be reviewed")
			assert.Nil(t, h.decisions.created, "nothing is recorded")
			assert.Zero(t, recorder.updated, "the proposal stays pending")
		})
	}
}

func TestDecide_ATaintedInternalApprovalWithoutADigestIsRefused(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{digest: strings.Repeat("d", 64)})
	h.proposals.proposal.Tainted = true

	_, err := decideWith(t, h, agentdecision.DecisionAccepted, "")

	require.Error(t, err)
	assert.True(t, agent.IsReviewRequired(err))
	assert.Nil(t, h.decisions.created)
}

func TestDecide_AGatedApprovalWithTheDigestShownIsRecordedReviewed(t *testing.T) {
	t.Parallel()

	digest := strings.Repeat("c", 64)
	h, _ := previewHarness(t, &fakePreviews{digest: digest})
	h.proposals.proposal.EgressClass = agent.EgressMoney

	_, err := decideWith(t, h, agentdecision.DecisionAccepted, digest)
	require.NoError(t, err)

	require.NotNil(t, h.decisions.created)
	assert.Equal(t, digest, h.decisions.created.PreviewDigest)
	assert.True(t, h.decisions.created.PreviewReviewed)
}

func TestDecide_AGatedRejectionNeedsNoDigest(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{digest: strings.Repeat("d", 64)})
	h.proposals.proposal.EgressClass = agent.EgressExternalRecipient
	h.proposals.proposal.Tainted = true

	_, err := decideWith(t, h, agentdecision.DecisionRejected, "")
	require.NoError(t, err, "a person can always say no")
	require.NotNil(t, h.decisions.created)
}

func TestDecide_AGatedApprovalWithNothingToPreviewItIsRefused(t *testing.T) {
	t.Parallel()

	h, _ := previewHarness(t, &fakePreviews{digest: strings.Repeat("d", 64)})
	h.service.previews = nil
	h.proposals.proposal.EgressClass = agent.EgressDriverVisible

	_, err := decideWith(t, h, agentdecision.DecisionAccepted, strings.Repeat("d", 64))

	require.Error(t, err)
	assert.True(t, agent.IsReviewRequired(err))
}

func planStep(
	h *decideHarness,
	step *agentpreview.ProposalPreview,
	reviewed bool,
) *agentdecisionports.DecideAgentProposalRequest {
	return &agentdecisionports.DecideAgentProposalRequest{
		ProposalID:          h.proposals.proposal.ID,
		Decision:            agentdecision.DecisionAccepted,
		ReasonCode:          "looks_right",
		WithinPlan:          true,
		StepPreview:         step,
		StepPreviewReviewed: reviewed,
		TenantInfo: pagination.TenantInfo{
			OrgID: h.actor.OrganizationID,
			BuID:  h.actor.BusinessUnitID,
		},
	}
}

func TestDecide_APlanStepIsReviewedByClass(t *testing.T) {
	t.Parallel()

	shown := &agentpreview.ProposalPreview{Digest: strings.Repeat("b", 64)}
	cases := []struct {
		name     string
		class    agent.EgressClass
		step     *agentpreview.ProposalPreview
		reviewed bool
		refused  bool
	}{
		{name: "an internal step of a plan decided unseen", class: agent.EgressInternal},
		{name: "a money step of a plan decided unseen", class: agent.EgressMoney, refused: true},
		{
			name:    "a money step whose plan preview was not shown",
			class:   agent.EgressMoney,
			step:    shown,
			refused: true,
		},
		{
			name:     "a money step whose plan preview was shown",
			class:    agent.EgressMoney,
			step:     shown,
			reviewed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h, _ := previewHarness(t, &fakePreviews{digest: strings.Repeat("d", 64)})
			h.proposals.proposal.EgressClass = tc.class

			_, err := h.service.DecideWithOutcome(
				t.Context(),
				planStep(h, tc.step, tc.reviewed),
				h.actor,
			)
			if tc.refused {
				require.Error(t, err)
				assert.True(t, agent.IsReviewRequired(err))
				assert.Nil(t, h.decisions.created)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, h.decisions.created)
			assert.Equal(t, tc.reviewed, h.decisions.created.PreviewReviewed)
		})
	}
}

func TestApproveDeferred_RefusesOnlyTheGatedRowsSentWithoutADigest(t *testing.T) {
	t.Parallel()

	h := newWindowHarness(t, 2)
	h.proposals.byID[h.ids[1]].EgressClass = agent.EgressMoney

	approved := h.approve(t)

	require.Len(t, approved, 2)
	require.NoError(t, approved[0].Err, "an internal write may still be approved unseen")
	require.Error(t, approved[1].Err)
	assert.True(t, agent.IsReviewRequired(approved[1].Err))
	assert.Equal(t, agent.ProposalStatusApproving, h.proposals.status(h.ids[0]))
	assert.Equal(t, agent.ProposalStatusPending, h.proposals.status(h.ids[1]))
}

func TestNeedsReview_AnEditThatChangesWhatTheCallReachesIsReviewed(t *testing.T) {
	t.Parallel()

	internal := &agent.AgentProposal{EgressClass: agent.EgressInternal, ToolName: "unknown_tool"}

	assert.False(t, (&Service{}).needsReview(internal, nil))
	assert.True(t, (&Service{}).needsReview(internal, map[string]any{"visibility": "Customer"}),
		"without the tool to classify the edit, it is reviewed")

	h := newDecideHarness(t)
	assert.True(t, h.service.needsReview(internal, map[string]any{"note": "x"}),
		"a tool the executor does not hold cannot be classified")

	known := *h.proposals.proposal
	assert.False(t, h.service.needsReview(&known, map[string]any{"note": "x"}),
		"an edit that keeps an internal write internal needs no preview")
}

type capturingAudit struct {
	auditports.AuditService

	logged []*auditports.LogActionParams
}

func (a *capturingAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	a.logged = append(a.logged, params)

	return nil
}

func TestDecide_TheAuditRowRecordsThePermissionTheDeciderUsed(t *testing.T) {
	t.Parallel()

	h := newDecideHarness(t)
	audit := &capturingAudit{}
	h.service.audit = audit

	outcome := h.decide(t, t.Context(), agentdecision.DecisionAccepted)
	require.NoError(t, outcome.ExecutionError)

	var decided *auditports.LogActionParams
	for _, row := range audit.logged {
		if row.Resource == permission.ResourceAgentProposal {
			decided = row
		}
	}
	require.NotNil(t, decided)
	assert.Equal(t, permission.OpUpdate, decided.Operation,
		"deciding is gated on agent_proposal:update, so the row records update")
	assert.True(t, decided.Critical)
}
