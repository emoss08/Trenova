package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTenderChanges struct {
	change  *editenderchange.TenderChange
	guard   *agenttoolstest.WriteGuard
	applied *editenderchangeservice.TenderChangeActionRequest
	refused *editenderchangeservice.TenderChangeActionRequest
}

func (f *fakeTenderChanges) GetTenderChange(
	_ context.Context,
	req editenderports.GetEDITenderChangeByIDRequest,
) (*editenderchange.TenderChange, error) {
	if req.ID != f.change.ID {
		return nil, errortypes.NewNotFoundError("EDITenderChange not found")
	}
	copied := *f.change

	return &copied, nil
}

func (f *fakeTenderChanges) ApplyTenderChange(
	_ context.Context,
	req *editenderchangeservice.TenderChangeActionRequest,
	_ *permissionports.RequestActor,
) (*editenderchange.TenderChange, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change, nil
}

func (f *fakeTenderChanges) RejectTenderChange(
	_ context.Context,
	req *editenderchangeservice.TenderChangeActionRequest,
	_ *permissionports.RequestActor,
) (*editenderchange.TenderChange, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.refused = req

	return f.change, nil
}

type fakeTransferChanges struct {
	change  *edisync.TransferChange
	effect  *editransferchangeservice.TransferChangeEffect
	checked error
	guard   *agenttoolstest.WriteGuard
	applied *editransferchangeservice.TransferChangeActionRequest
	refused *editransferchangeservice.TransferChangeActionRequest
}

func (f *fakeTransferChanges) GetTransferChange(
	_ context.Context,
	req editenderports.GetEDITransferChangeByIDRequest,
) (*edisync.TransferChange, error) {
	if req.ID != f.change.ID {
		return nil, errortypes.NewNotFoundError("EDITransferChange not found")
	}
	copied := *f.change

	return &copied, nil
}

func (f *fakeTransferChanges) CheckTransferChangeReview(
	_ context.Context,
	change *edisync.TransferChange,
	_ pagination.TenantInfo,
) error {
	if change.Status != edisync.TransferChangeStatusPendingReview {
		return errortypes.NewValidationError(
			"status", errortypes.ErrInvalidOperation,
			"EDI transfer change has already been reviewed",
		)
	}

	return f.checked
}

func (f *fakeTransferChanges) PlanTransferChangeEffect(
	context.Context,
	pagination.TenantInfo,
	*edisync.TransferChange,
) (*editransferchangeservice.TransferChangeEffect, error) {
	if f.effect == nil {
		return &editransferchangeservice.TransferChangeEffect{}, nil
	}

	return f.effect, nil
}

func (f *fakeTransferChanges) ApplyTransferChange(
	_ context.Context,
	req *editransferchangeservice.TransferChangeActionRequest,
	_ *permissionports.RequestActor,
) (*edisync.TransferChange, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change, nil
}

func (f *fakeTransferChanges) RejectTransferChange(
	_ context.Context,
	req *editransferchangeservice.TransferChangeActionRequest,
	_ *permissionports.RequestActor,
) (*edisync.TransferChange, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.refused = req

	return f.change, nil
}

func pendingTenderChange(orgID, buID pulid.ID) *editenderchange.TenderChange {
	weightBefore, weightAfter := int64(20000), int64(42000)

	return &editenderchange.TenderChange{
		ID:            pulid.MustNew("editcg_"),
		Status:        editenderchange.TenderChangeStatusPendingReview,
		RecipientKind: editenderchange.TenderRecipientKindInternal,
		Version:       2,
		Recipient: &editenderchange.TenderRecipient{
			RecipientKind:           editenderchange.TenderRecipientKindInternal,
			RecipientOrganizationID: orgID,
			BusinessUnitID:          buID,
		},
		PreviousBaselinePayload: editransfer.LoadTenderPayload{BOL: "BOL-9", Weight: &weightBefore},
		NewTenderPayload:        editransfer.LoadTenderPayload{BOL: "BOL-9", Weight: &weightAfter},
	}
}

func TestEDIChangeReviews_ArePersonOnlyProposalsTargetingTheChange(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		NewReviewEDITenderChangeTool(nil),
		NewReviewEDITransferChangeTool(nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, permission.ResourceEDI, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(
			t,
			[]agent.EgressClass{agent.EgressExternalRecipient},
			policy.Egress,
			tool.Name(),
		)

		id := pulid.MustNew("editcg_")
		target, ok := tool.(agenttoolports.TargetedTool).Target(
			map[string]any{paramEDIChangeID: id.String()},
		)
		require.True(t, ok, tool.Name())
		assert.Equal(t, agenttoolports.ToolTarget{Resource: permission.ResourceEDI, ID: id}, target)
	}
}

func TestReviewEDITenderChange_PreviewsWhatTheTenderNowSaysAndAppliesOnlyWhenApproved(
	t *testing.T,
) {
	t.Parallel()

	params := agenttoolstest.ApprovedParams(nil)
	changes := &fakeTenderChanges{
		change: pendingTenderChange(params.OrganizationID, params.BusinessUnitID),
		guard:  &agenttoolstest.WriteGuard{},
	}
	tool := NewReviewEDITenderChangeTool(changes).(*EdiChangeReviewTool)
	args := map[string]any{
		paramEDIChangeID:       changes.change.ID.String(),
		paramEDIReviewDecision: reviewApply,
	}
	params.Params = args
	proposing := *params
	proposing.ProposalID = pulid.Nil

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		changes.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), &proposing)
		},
	)
	status := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, changes.change.ID, status.EntityID)
	require.NotNil(t, status.Version)
	assert.Equal(t, int64(2), *status.Version)
	assert.Equal(t, "Applied", agenttoolstest.FieldByPath(t, status, "status").After)
	tender := agenttoolstest.PreviewChange(t, preview, 1)
	weight := agenttoolstest.FieldByPath(t, tender, "weight")
	assert.EqualValues(t, 20000, weight.Before)
	assert.EqualValues(t, 42000, weight.After)

	require.NoError(t, tool.Validate(t.Context(), &proposing))
	require.ErrorIs(t, tool.Execute(t.Context(), &proposing), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, changes.applied)
	assert.Equal(t, changes.change.ID, changes.applied.ChangeID)
	assert.Nil(t, changes.refused)
}

func TestReviewEDITenderChange_RefusesWhatTheServiceRefuses(t *testing.T) {
	t.Parallel()

	params := agenttoolstest.AgentParamsFor(nil)
	cases := map[string]func(*editenderchange.TenderChange){
		"already reviewed": func(change *editenderchange.TenderChange) {
			change.Status = editenderchange.TenderChangeStatusApplied
		},
		"another organization's change": func(change *editenderchange.TenderChange) {
			change.Recipient.RecipientOrganizationID = pulid.MustNew("org_")
		},
		"an outside partner's change": func(change *editenderchange.TenderChange) {
			change.Recipient.RecipientKind = editenderchange.TenderRecipientKindExternal
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			change := pendingTenderChange(params.OrganizationID, params.BusinessUnitID)
			mutate(change)
			fake := &fakeTenderChanges{change: change, guard: &agenttoolstest.WriteGuard{}}
			tool := NewReviewEDITenderChangeTool(fake).(*EdiChangeReviewTool)
			call := *params
			call.Params = map[string]any{
				paramEDIChangeID:       change.ID.String(),
				paramEDIReviewDecision: reviewReject,
			}

			require.Error(t, tool.Validate(t.Context(), &call))
			preview := agenttoolstest.PreviewWithoutWrites(
				t,
				fake.guard,
				func() (*agentpreview.ToolPreview, error) {
					return tool.Preview(t.Context(), &call)
				},
			)
			agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
		})
	}
}

func TestReviewEDITransferChange_PreviewsTheShipmentStatusItWouldSet(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	changes := &fakeTransferChanges{
		change: &edisync.TransferChange{
			ID:         pulid.MustNew("editc_"),
			Status:     edisync.TransferChangeStatusPendingReview,
			ChangeType: edisync.TransferChangeTypeShipmentStatus214,
			Version:    6,
		},
		effect: &editransferchangeservice.TransferChangeEffect{
			Shipment: &shipment.Shipment{
				ID:        shipmentID,
				ProNumber: "PRO-31",
				Status:    shipmentvocab.StatusInTransit,
				Version:   9,
			},
			NextStatus: shipmentvocab.StatusCompleted,
		},
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewReviewEDITransferChangeTool(changes).(*EdiChangeReviewTool)
	args := map[string]any{
		paramEDIChangeID:       changes.change.ID.String(),
		paramEDIReviewDecision: reviewApply,
	}

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		changes.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	assert.Equal(
		t,
		"Applied",
		agenttoolstest.FieldByPath(t, agenttoolstest.PreviewChange(t, preview, 0), "status").After,
	)
	moved := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, moved.Resource)
	assert.Equal(t, shipmentID, moved.EntityID)
	assert.Equal(t, "Completed", agenttoolstest.FieldByPath(t, moved, "status").After)

	rejected := agenttoolstest.AgentParamsFor(map[string]any{
		paramEDIChangeID:          changes.change.ID.String(),
		paramEDIReviewDecision:    reviewReject,
		changepreview.FieldReason: "The partner's 214 contradicts the driver's arrival",
	})
	rejectPreview := agenttoolstest.PreviewWithoutWrites(
		t,
		changes.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), rejected)
		},
	)
	require.Len(t, rejectPreview.Changes, 1, "a rejection leaves the shipment alone")

	approved := *rejected
	approved.ProposalID = pulid.MustNew("agp_")
	require.NoError(t, tool.Execute(t.Context(), &approved))
	require.NotNil(t, changes.refused)
	assert.Equal(t, "The partner's 214 contradicts the driver's arrival", changes.refused.Reason)
}

func TestReviewEDIChanges_RefuseAnUnknownDecision(t *testing.T) {
	t.Parallel()

	tool := NewReviewEDITransferChangeTool(&fakeTransferChanges{
		change: &edisync.TransferChange{ID: pulid.MustNew("editc_")},
	})
	err := tool.(agenttoolports.ToolValidator).Validate(
		t.Context(),
		agenttoolstest.AgentParamsFor(map[string]any{
			paramEDIChangeID:       pulid.MustNew("editc_").String(),
			paramEDIReviewDecision: "Approve",
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Apply or Reject")
}
