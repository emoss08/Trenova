package findingtools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/detentionpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/agentceiling"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentioncollectability"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubApprover struct {
	occurrence *detention.DetentionOccurrence
	approved   *detentionreview.ApproveParams
	previewed  *detentionreview.ApproveParams
}

func (s *stubApprover) PreviewApprove(
	_ context.Context,
	p *detentionreview.ApproveParams,
) (*detentionreview.OccurrenceChange, error) {
	s.previewed = p
	before := *s.occurrence
	after := before
	if err := after.Approve(p.UserID, 1_767_230_000); err != nil {
		return nil, err
	}

	return &detentionreview.OccurrenceChange{Before: &before, After: &after}, nil
}

func (s *stubApprover) Approve(
	_ context.Context,
	p *detentionreview.ApproveParams,
) (*detention.DetentionOccurrence, error) {
	s.approved = p

	return s.occurrence, nil
}

func (s *stubApprover) GetOccurrenceDetail(
	context.Context,
	*detentionports.GetDetentionOccurrenceByIDRequest,
) (*detentionreview.OccurrenceDetail, error) {
	return &detentionreview.OccurrenceDetail{
		Occurrence: s.occurrence,
		Evidence:   []*detentionevidence.DetentionEvidence{{}, {}},
		Collectability: detentioncollectability.CollectabilityAssessment{
			Score: 88,
			Band:  "Strong",
		},
	}, nil
}

func heldOccurrence(status detention.OccurrenceStatus) *detention.DetentionOccurrence {
	return &detention.DetentionOccurrence{
		ID:               pulid.MustNew("dto_"),
		Status:           status,
		RequiresApproval: true,
		BillableAmount:   decimal.NewFromInt(425),
		Currency:         "USD",
	}
}

func approveParams(occurrenceID pulid.ID) agenttoolports.ToolExecuteParams {
	params := agenttoolstest.DeskParams(map[string]any{
		"occurrenceId": occurrenceID.String(),
		"evidence": " Arrival 08:02 and departure 12:40 on the ELD; notice sent 09:55, " +
			"inside the window. ",
	})
	params.ProposalID = pulid.MustNew("agp_")

	return params
}

func TestApproveDetention_ApprovesAsThePersonWhoApprovedTheProposal(t *testing.T) {
	t.Parallel()

	stub := &stubApprover{occurrence: heldOccurrence(detention.OccurrenceStatusPending)}
	tool := NewApproveDetentionTool(stub)
	params := approveParams(stub.occurrence.ID)

	require.NoError(t, tool.(agenttoolports.ToolValidator).Validate(t.Context(), params))

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, stub.approved, "a preview must not approve")
	assert.Contains(t, preview.Summary, "425.00 USD")
	assert.Contains(t, preview.Summary, "collectability 88, Strong; 2 evidence records")
	assert.Contains(t, preview.Summary, "notice sent 09:55")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, stub.approved)
	assert.Equal(t, *stub.previewed, *stub.approved,
		"the preview and the write must make the same approval")
	assert.Equal(t, stub.occurrence.ID, stub.approved.OccurrenceID)
	assert.Equal(t, params.Actor.UserID, stub.approved.UserID)
	assert.Equal(t, params.OrganizationID, stub.approved.TenantInfo.OrgID)
	assert.Equal(t,
		"Arrival 08:02 and departure 12:40 on the ELD; notice sent 09:55, inside the window.",
		stub.approved.Note,
	)
}

func TestApproveDetention_RefusesAnythingButAPendingCharge(t *testing.T) {
	t.Parallel()

	for _, status := range []detention.OccurrenceStatus{
		detention.OccurrenceStatusAccruing,
		detention.OccurrenceStatusApproved,
		detention.OccurrenceStatusBilled,
		detention.OccurrenceStatusWaived,
		detention.OccurrenceStatusDisputed,
		detention.OccurrenceStatusNotBillable,
	} {
		stub := &stubApprover{occurrence: heldOccurrence(status)}
		tool := NewApproveDetentionTool(stub)
		params := approveParams(stub.occurrence.ID)

		err := tool.(agenttoolports.ToolValidator).Validate(t.Context(), params)
		require.Error(t, err, status)
		assert.Contains(t, err.Error(), "only a pending charge can be approved", status)
		assert.Contains(t, err.Error(), string(status), status)

		require.Error(t, tool.Execute(t.Context(), params), status)
		assert.Nil(t, stub.approved, status)
	}
}

func TestApproveDetention_NeverRunsWithoutAProposalAPersonApproved(t *testing.T) {
	t.Parallel()

	stub := &stubApprover{occurrence: heldOccurrence(detention.OccurrenceStatusPending)}
	tool := NewApproveDetentionTool(stub)
	params := approveParams(stub.occurrence.ID)
	params.ProposalID = pulid.Nil

	require.ErrorIs(t, tool.Execute(t.Context(), params), detentionpreview.ErrApprovalNeedsAPerson)
	assert.Nil(t, stub.approved)
}

func TestApproveDetention_IsAMoneyWriteCappedAtPropose(t *testing.T) {
	t.Parallel()

	policy := NewApproveDetentionTool(&stubApprover{}).Policy()

	assert.Equal(t, permission.ResourceDetention, policy.Resource)
	assert.Equal(t, permission.OpUpdate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.DefaultTier)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, policy.Egress)
	assert.False(t, policy.Reversible)
	assert.True(t, agentceiling.Allows(policy.Resource, policy.Operation))
}

func TestApproveDetention_RequiresTheEvidence(t *testing.T) {
	t.Parallel()

	stub := &stubApprover{occurrence: heldOccurrence(detention.OccurrenceStatusPending)}
	tool := NewApproveDetentionTool(stub)
	params := approveParams(stub.occurrence.ID)
	delete(params.Params, "evidence")

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, stub.approved)
}
