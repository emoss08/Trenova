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
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTransfers struct {
	transfer   *edi.EDITransfer
	unresolved []edi.MappingResolution
	guard      *writeGuard
	directions []string
	approved   *ediservice.ApproveTransferRequest
	rejected   *ediservice.RejectTransferRequest
	canceled   *ediservice.CancelTransferRequest
	expired    *ediservice.ExpireTransferRequest
	actor      *serviceports.RequestActor
}

func (f *fakeTransfers) GetTransfer(
	_ context.Context,
	req repositories.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	f.directions = append(f.directions, req.Direction)
	if req.ID != f.transfer.ID {
		return nil, errortypes.NewNotFoundError("EDITenderTransfer not found")
	}
	copied := *f.transfer

	return &copied, nil
}

func (f *fakeTransfers) PlanApproveTransfer(
	ctx context.Context,
	req *ediservice.ApproveTransferRequest,
	actor *serviceports.RequestActor,
) (*ediservice.TransferApprovalPlan, error) {
	transfer, err := f.GetTransfer(ctx, repositories.GetEDITransferByIDRequest{
		ID:        req.TransferID,
		Direction: ediservice.TransferDirectionInbound,
	})
	if err != nil {
		return nil, err
	}

	plan := &ediservice.TransferApprovalPlan{
		Transfer:      transfer,
		SendsResponse: transfer.InboundMessageID.IsNotNil(),
		Mapping:       &ediservice.MappingPreview{Unresolved: f.unresolved},
	}
	if plan.Refusal = ediservice.RequireActionableTransfer(transfer, "approved"); plan.Refusal != nil {
		return plan, nil
	}
	if len(f.unresolved) > 0 {
		plan.Refusal = errortypes.NewValidationError(
			"Customer", errortypes.ErrRequired, "Mapping is required",
		)
		return plan, nil
	}

	after := *transfer
	ediservice.MarkTransferApprovalStarted(&after, &ediservice.TransferApprovalMark{
		BusinessUnitID: req.TenantInfo.BuID,
		ApproverID:     actorUserID(actor),
		At:             timeutils.NowUnix(),
	})
	plan.After = &after
	plan.Shipment = &shipment.Shipment{
		BOL:         transfer.TenderPayload.BOL,
		CustomerID:  pulid.MustNew("cus_"),
		Status:      shipment.StatusNew,
		EntryMethod: shipment.EntryMethodEDI,
	}

	return plan, nil
}

func (f *fakeTransfers) ApproveTransfer(
	_ context.Context,
	req *ediservice.ApproveTransferRequest,
	actor *serviceports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.approved, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) RejectTransfer(
	_ context.Context,
	req *ediservice.RejectTransferRequest,
	actor *serviceports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.rejected, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) CancelTransfer(
	_ context.Context,
	req *ediservice.CancelTransferRequest,
	actor *serviceports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.canceled, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) ExpireTransfer(
	_ context.Context,
	req *ediservice.ExpireTransferRequest,
	actor *serviceports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.expired, f.actor = req, actor

	return f.transfer, nil
}

func inboundTender(status edi.TransferStatus) *edi.EDITransfer {
	partner := &edi.EDIPartner{Name: "Acme Shipper"}

	return &edi.EDITransfer{
		ID:               pulid.MustNew("edilt_"),
		Status:           status,
		InboundMessageID: pulid.MustNew("edimsg_"),
		TargetPartnerID:  pulid.MustNew("edip_"),
		TargetPartner:    partner,
		SourcePartner:    partner,
		Version:          4,
		TenderPayload: edi.LoadTenderPayload{
			BOL:           "BOL-77",
			CustomerLabel: "Acme Foods",
			Moves: []edi.LoadTenderMove{{Stops: []edi.LoadTenderStop{
				{Type: "Pickup"}, {Type: "Delivery"},
			}}},
		},
	}
}

func tenderParams(transfer *edi.EDITransfer, extra map[string]any) map[string]any {
	params := map[string]any{paramEDITransferID: transfer.ID.String()}
	for key, value := range extra {
		params[key] = value
	}

	return params
}

func TestEDITenderDecisions_ArePersonOnlyProposalsOnTheEDIResource(t *testing.T) {
	t.Parallel()

	for _, tool := range []serviceports.AgentTool{
		newAcceptEDILoadTenderTool(nil),
		newDeclineEDILoadTenderTool(nil),
		newCancelEDILoadTenderTool(nil),
		newExpireEDILoadTenderTool(nil),
	} {
		policy := tool.Policy()
		assert.Equal(t, permission.ResourceEDI, policy.Resource, tool.Name())
		assert.Equal(t, permission.OpUpdate, policy.Operation, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.DefaultTier, tool.Name())
		assert.Equal(t, agent.TierPropose, policy.MaxTier, tool.Name())
		assert.Equal(
			t,
			[]agent.EgressClass{agent.EgressExternalRecipient},
			policy.Egress,
			tool.Name(),
		)

		id := pulid.MustNew("edilt_")
		target, ok := tool.(serviceports.TargetedTool).Target(
			map[string]any{paramEDITransferID: id.String()},
		)
		require.True(t, ok, tool.Name())
		assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceEDI, ID: id}, target)
	}
}

func TestAcceptEDILoadTender_PreviewsTheShipmentAndThe990ThenRunsOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(edi.TransferStatusPendingApproval),
		guard:    &writeGuard{},
	}
	tool := newAcceptEDILoadTenderTool(transfers).(*ediTransferDecisionTool)
	args := tenderParams(transfers.transfer, nil)

	preview := previewWithoutWrites(t, transfers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	status := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceEDI, status.Resource)
	assert.Equal(t, transfers.transfer.ID, status.EntityID)
	require.NotNil(t, status.Version)
	assert.Equal(t, int64(4), *status.Version)
	assert.Equal(t, "Processing", fieldByPath(t, status, "status").After)

	created := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, created.Resource)
	assert.Equal(t, agent.PreviewOperationCreate, created.Operation)

	response := previewChange(t, preview, len(preview.Changes)-1)
	require.NotNil(t, response.Message)
	assert.Equal(t, agent.MessageChannelEDI, response.Message.Channel)
	assert.Contains(t, response.Message.Body, "Accepted (A)")
	assert.Contains(t, response.Message.Body, "BOL-77")
	assert.Contains(t, preview.Summary, "Acme Foods")

	require.NoError(t, tool.Validate(t.Context(), agentParamsFor(args)))
	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	assert.Nil(t, transfers.approved, "an agent never accepts a tender on its own")

	approved := approvedParams(args)
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, transfers.approved)
	assert.Equal(t, transfers.transfer.ID, transfers.approved.TransferID)
	assert.Equal(t, approved.OrganizationID, transfers.approved.TenantInfo.OrgID)
	assert.Equal(t, approved.Actor, transfers.actor)
}

func TestAcceptEDILoadTender_RefusesATenderWhoseMappingsAreIncomplete(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer:   inboundTender(edi.TransferStatusMappingRequired),
		unresolved: []edi.MappingResolution{{EntityType: edi.MappingEntityTypeCustomer}},
		guard:      &writeGuard{},
	}
	tool := newAcceptEDILoadTenderTool(transfers).(*ediTransferDecisionTool)
	args := agentParamsFor(tenderParams(transfers.transfer, nil))

	require.Error(t, tool.Validate(t.Context(), args))
	preview := previewWithoutWrites(t, transfers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), args)
	})
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)
}

func TestDeclineEDILoadTender_CarriesTheReasonToThePartner(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(edi.TransferStatusPendingApproval),
		guard:    &writeGuard{},
	}
	tool := newDeclineEDILoadTenderTool(transfers).(*ediTransferDecisionTool)

	missing := agentParamsFor(tenderParams(transfers.transfer, nil))
	require.Error(t, tool.Validate(t.Context(), missing), "a decline needs a reason")

	args := tenderParams(transfers.transfer, map[string]any{
		fieldReason: "No reefer capacity on that lane this week",
	})
	preview := previewWithoutWrites(t, transfers.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	status := previewChange(t, preview, 0)
	assert.Equal(t, "Rejected", fieldByPath(t, status, "status").After)
	assert.Equal(t, "No reefer capacity on that lane this week",
		fieldByPath(t, status, "rejectionReason").After)
	response := previewChange(t, preview, 1)
	require.NotNil(t, response.Message)
	assert.Contains(t, response.Message.Body, "Declined (D)")
	assert.Contains(t, response.Message.Body, "No reefer capacity on that lane this week")
	assert.Equal(t, []string{ediservice.TransferDirectionInbound}, transfers.directions)

	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(args)))
	require.NotNil(t, transfers.rejected)
	assert.Equal(t, "No reefer capacity on that lane this week", transfers.rejected.Reason)
}

func TestCancelAndExpireEDILoadTender_ReadTheSideTheServiceActsOn(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		build     func(ediTransferDecider) serviceports.AgentTool
		direction string
		status    string
		wrote     func(*fakeTransfers) bool
	}{
		"cancel": {
			build:     newCancelEDILoadTenderTool,
			direction: ediservice.TransferDirectionOutbound,
			status:    "Canceled",
			wrote:     func(f *fakeTransfers) bool { return f.canceled != nil },
		},
		"expire": {
			build:     newExpireEDILoadTenderTool,
			direction: ediservice.TransferDirectionAny,
			status:    "Expired",
			wrote:     func(f *fakeTransfers) bool { return f.expired != nil },
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			transfers := &fakeTransfers{
				transfer: inboundTender(edi.TransferStatusPendingApproval),
				guard:    &writeGuard{},
			}
			transfers.transfer.InboundMessageID = pulid.Nil
			tool := tc.build(transfers).(*ediTransferDecisionTool)
			args := tenderParams(transfers.transfer, nil)

			preview := previewWithoutWrites(t, transfers.guard, func() (*agent.ToolPreview, error) {
				return tool.Preview(t.Context(), agentParamsFor(args))
			})
			require.Len(t, preview.Changes, 1, "a tender between organizations sends no 990")
			assert.Equal(t, tc.status, fieldByPath(t, previewChange(t, preview, 0), "status").After)
			assert.Equal(t, []string{tc.direction}, transfers.directions)

			require.NoError(t, tool.Execute(t.Context(), approvedParams(args)))
			assert.True(t, tc.wrote(transfers))
		})
	}
}

func TestEDITenderDecisions_RefuseAFinalizedTransfer(t *testing.T) {
	t.Parallel()

	for _, build := range []func(ediTransferDecider) serviceports.AgentTool{
		newAcceptEDILoadTenderTool,
		newDeclineEDILoadTenderTool,
		newCancelEDILoadTenderTool,
		newExpireEDILoadTenderTool,
	} {
		transfers := &fakeTransfers{
			transfer: inboundTender(edi.TransferStatusApproved),
			guard:    &writeGuard{},
		}
		tool := build(transfers).(*ediTransferDecisionTool)
		args := agentParamsFor(tenderParams(transfers.transfer, map[string]any{
			fieldReason: "Duplicate of a tender already accepted",
		}))

		err := tool.Validate(t.Context(), args)
		require.Error(t, err, tool.Name())
		assert.Contains(t, err.Error(), "finalized or processing", tool.Name())

		preview := previewWithoutWrites(t, transfers.guard, func() (*agent.ToolPreview, error) {
			return tool.Preview(t.Context(), args)
		})
		requireWarning(t, preview, agent.PreviewWarningWouldFail)
	}
}

func TestEDITenderDecisions_RefuseAnActorFromAnotherTenant(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(edi.TransferStatusPendingApproval),
		guard:    &writeGuard{},
	}
	tool := newAcceptEDILoadTenderTool(transfers)
	params := approvedParams(tenderParams(transfers.transfer, nil))
	params.OrganizationID = pulid.MustNew("org_")

	require.ErrorIs(t, tool.Execute(t.Context(), params), ErrTenantMismatch)
	assert.Nil(t, transfers.approved)
}
