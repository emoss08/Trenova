package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTenderChanges struct {
	change  *edi.TenderChange
	guard   *writeGuard
	applied *ediservice.TenderChangeActionRequest
	refused *ediservice.TenderChangeActionRequest
}

func (f *fakeTenderChanges) GetTenderChange(
	_ context.Context,
	req repositories.GetEDITenderChangeByIDRequest,
) (*edi.TenderChange, error) {
	if req.ID != f.change.ID {
		return nil, errortypes.NewNotFoundError("EDITenderChange not found")
	}
	copied := *f.change

	return &copied, nil
}

func (f *fakeTenderChanges) ApplyTenderChange(
	_ context.Context,
	req *ediservice.TenderChangeActionRequest,
	_ *serviceports.RequestActor,
) (*edi.TenderChange, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change, nil
}

func (f *fakeTenderChanges) RejectTenderChange(
	_ context.Context,
	req *ediservice.TenderChangeActionRequest,
	_ *serviceports.RequestActor,
) (*edi.TenderChange, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.refused = req

	return f.change, nil
}

type fakeTransferChanges struct {
	change  *edi.TransferChange
	effect  *ediservice.TransferChangeEffect
	checked error
	guard   *writeGuard
	applied *ediservice.TransferChangeActionRequest
	refused *ediservice.TransferChangeActionRequest
}

func (f *fakeTransferChanges) GetTransferChange(
	_ context.Context,
	req repositories.GetEDITransferChangeByIDRequest,
) (*edi.TransferChange, error) {
	if req.ID != f.change.ID {
		return nil, errortypes.NewNotFoundError("EDITransferChange not found")
	}
	copied := *f.change

	return &copied, nil
}

func (f *fakeTransferChanges) CheckTransferChangeReview(
	_ context.Context,
	change *edi.TransferChange,
	_ pagination.TenantInfo,
) error {
	if change.Status != edi.TransferChangeStatusPendingReview {
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
	*edi.TransferChange,
) (*ediservice.TransferChangeEffect, error) {
	if f.effect == nil {
		return &ediservice.TransferChangeEffect{}, nil
	}

	return f.effect, nil
}

func (f *fakeTransferChanges) ApplyTransferChange(
	_ context.Context,
	req *ediservice.TransferChangeActionRequest,
	_ *serviceports.RequestActor,
) (*edi.TransferChange, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.applied = req

	return f.change, nil
}

func (f *fakeTransferChanges) RejectTransferChange(
	_ context.Context,
	req *ediservice.TransferChangeActionRequest,
	_ *serviceports.RequestActor,
) (*edi.TransferChange, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.refused = req

	return f.change, nil
}

func pendingTenderChange(orgID, buID pulid.ID) *edi.TenderChange {
	weightBefore, weightAfter := int64(20000), int64(42000)

	return &edi.TenderChange{
		ID:            pulid.MustNew("editcg_"),
		Status:        edi.TenderChangeStatusPendingReview,
		RecipientKind: edi.TenderRecipientKindInternal,
		Version:       2,
		Recipient: &edi.TenderRecipient{
			RecipientKind:           edi.TenderRecipientKindInternal,
			RecipientOrganizationID: orgID,
			RecipientBusinessUnitID: buID,
		},
		PreviousBaselinePayload: edi.LoadTenderPayload{BOL: "BOL-9", Weight: &weightBefore},
		NewTenderPayload:        edi.LoadTenderPayload{BOL: "BOL-9", Weight: &weightAfter},
	}
}

func TestEDIChangeReviews_ArePersonOnlyProposalsTargetingTheChange(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentTool{
		newReviewEDITenderChangeTool(nil),
		newReviewEDITransferChangeTool(nil),
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
		target, ok := tool.(serviceports.TargetedTool).Target(
			map[string]any{paramEDIChangeID: id.String()},
		)
		require.True(t, ok, tool.Name())
		assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceEDI, ID: id}, target)
	}
}

func TestReviewEDITenderChange_PreviewsWhatTheTenderNowSaysAndAppliesOnlyWhenApproved(
	t *testing.T,
) {
	t.Parallel()

	params := approvedParams(nil)
	changes := &fakeTenderChanges{
		change: pendingTenderChange(params.OrganizationID, params.BusinessUnitID),
		guard:  &writeGuard{},
	}
	tool := newReviewEDITenderChangeTool(changes).(*ediChangeReviewTool)
	args := map[string]any{
		paramEDIChangeID:       changes.change.ID.String(),
		paramEDIReviewDecision: reviewApply,
	}
	params.Params = args
	proposing := params
	proposing.ProposalID = pulid.Nil

	preview := previewWithoutWrites(t, changes.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), proposing)
	})
	status := previewChange(t, preview, 0)
	assert.Equal(t, changes.change.ID, status.EntityID)
	require.NotNil(t, status.Version)
	assert.Equal(t, int64(2), *status.Version)
	assert.Equal(t, "Applied", fieldByPath(t, status, "status").After)
	tender := previewChange(t, preview, 1)
	weight := fieldByPath(t, tender, "weight")
	assert.EqualValues(t, 20000, weight.Before)
	assert.EqualValues(t, 42000, weight.After)

	require.NoError(t, tool.Validate(t.Context(), proposing))
	require.ErrorIs(t, tool.Execute(t.Context(), proposing), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, changes.applied)
	assert.Equal(t, changes.change.ID, changes.applied.ChangeID)
	assert.Nil(t, changes.refused)
}

func TestReviewEDITenderChange_RefusesWhatTheServiceRefuses(t *testing.T) {
	t.Parallel()

	params := agentParamsFor(nil)
	cases := map[string]func(*edi.TenderChange){
		"already reviewed": func(change *edi.TenderChange) {
			change.Status = edi.TenderChangeStatusApplied
		},
		"another organization's change": func(change *edi.TenderChange) {
			change.Recipient.RecipientOrganizationID = pulid.MustNew("org_")
		},
		"an outside partner's change": func(change *edi.TenderChange) {
			change.Recipient.RecipientKind = edi.TenderRecipientKindExternal
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			change := pendingTenderChange(params.OrganizationID, params.BusinessUnitID)
			mutate(change)
			fake := &fakeTenderChanges{change: change, guard: &writeGuard{}}
			tool := newReviewEDITenderChangeTool(fake).(*ediChangeReviewTool)
			call := params
			call.Params = map[string]any{
				paramEDIChangeID:       change.ID.String(),
				paramEDIReviewDecision: reviewReject,
			}

			require.Error(t, tool.Validate(t.Context(), call))
			preview := previewWithoutWrites(t, fake.guard, func() (*agent.ToolPreview, error) {
				return tool.Preview(t.Context(), call)
			})
			requireWarning(t, preview, agent.PreviewWarningWouldFail)
		})
	}
}

func TestReviewEDITransferChange_PreviewsTheShipmentStatusItWouldSet(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	changes := &fakeTransferChanges{
		change: &edi.TransferChange{
			ID:         pulid.MustNew("editc_"),
			Status:     edi.TransferChangeStatusPendingReview,
			ChangeType: edi.TransferChangeTypeShipmentStatus214,
			Version:    6,
		},
		effect: &ediservice.TransferChangeEffect{
			Shipment: &shipment.Shipment{
				ID:        shipmentID,
				ProNumber: "PRO-31",
				Status:    shipment.StatusInTransit,
				Version:   9,
			},
			NextStatus: shipment.StatusCompleted,
		},
		guard: &writeGuard{},
	}
	tool := newReviewEDITransferChangeTool(changes).(*ediChangeReviewTool)
	args := map[string]any{
		paramEDIChangeID:       changes.change.ID.String(),
		paramEDIReviewDecision: reviewApply,
	}

	preview := previewWithoutWrites(t, changes.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	assert.Equal(t, "Applied", fieldByPath(t, previewChange(t, preview, 0), "status").After)
	moved := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, moved.Resource)
	assert.Equal(t, shipmentID, moved.EntityID)
	assert.Equal(t, "Completed", fieldByPath(t, moved, "status").After)

	rejected := agentParamsFor(map[string]any{
		paramEDIChangeID:       changes.change.ID.String(),
		paramEDIReviewDecision: reviewReject,
		fieldReason:            "The partner's 214 contradicts the driver's arrival",
	})
	rejectPreview := previewWithoutWrites(t, changes.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), rejected)
	})
	require.Len(t, rejectPreview.Changes, 1, "a rejection leaves the shipment alone")

	approved := rejected
	approved.ProposalID = pulid.MustNew("agp_")
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, changes.refused)
	assert.Equal(t, "The partner's 214 contradicts the driver's arrival", changes.refused.Reason)
}

func TestReviewEDIChanges_RefuseAnUnknownDecision(t *testing.T) {
	t.Parallel()

	tool := newReviewEDITransferChangeTool(&fakeTransferChanges{
		change: &edi.TransferChange{ID: pulid.MustNew("editc_")},
	})
	err := tool.(serviceports.ToolValidator).Validate(t.Context(), agentParamsFor(map[string]any{
		paramEDIChangeID:       pulid.MustNew("editc_").String(),
		paramEDIReviewDecision: "Approve",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Apply or Reject")
}
