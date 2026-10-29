package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTransfers struct {
	transfer   *editransfer.EDITransfer
	unresolved []editransfer.MappingResolution
	guard      *agenttoolstest.WriteGuard
	directions []string
	approved   *ediservice.ApproveTransferRequest
	rejected   *ediservice.RejectTransferRequest
	canceled   *ediservice.CancelTransferRequest
	expired    *ediservice.ExpireTransferRequest
	actor      *permissionports.RequestActor
}

func (f *fakeTransfers) GetTransfer(
	_ context.Context,
	req editenderports.GetEDITransferByIDRequest,
) (*editransfer.EDITransfer, error) {
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
	actor *permissionports.RequestActor,
) (*ediservice.TransferApprovalPlan, error) {
	transfer, err := f.GetTransfer(ctx, editenderports.GetEDITransferByIDRequest{
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
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.approved, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) RejectTransfer(
	_ context.Context,
	req *ediservice.RejectTransferRequest,
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.rejected, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) CancelTransfer(
	_ context.Context,
	req *ediservice.CancelTransferRequest,
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.canceled, f.actor = req, actor

	return f.transfer, nil
}

func (f *fakeTransfers) ExpireTransfer(
	_ context.Context,
	req *ediservice.ExpireTransferRequest,
	actor *permissionports.RequestActor,
) (*editransfer.EDITransfer, error) {
	if err := f.guard.Write(); err != nil {
		return nil, err
	}
	f.expired, f.actor = req, actor

	return f.transfer, nil
}

func inboundTender(status editransfer.TransferStatus) *editransfer.EDITransfer {
	partner := &edi.EDIPartner{Name: "Acme Shipper"}

	return &editransfer.EDITransfer{
		ID:               pulid.MustNew("edilt_"),
		Status:           status,
		InboundMessageID: pulid.MustNew("edimsg_"),
		TargetPartnerID:  pulid.MustNew("edip_"),
		TargetPartner:    partner,
		SourcePartner:    partner,
		Version:          4,
		TenderPayload: editransfer.LoadTenderPayload{
			BOL:           "BOL-77",
			CustomerLabel: "Acme Foods",
			Moves: []editransfer.LoadTenderMove{{Stops: []editransfer.LoadTenderStop{
				{Type: "Pickup"}, {Type: "Delivery"},
			}}},
		},
	}
}

func tenderParams(transfer *editransfer.EDITransfer, extra map[string]any) map[string]any {
	params := map[string]any{paramEDITransferID: transfer.ID.String()}
	for key, value := range extra {
		params[key] = value
	}

	return params
}

func TestEDITenderDecisions_ArePersonOnlyProposalsOnTheEDIResource(t *testing.T) {
	t.Parallel()

	for _, tool := range []agenttoolports.AgentTool{
		NewAcceptEDILoadTenderTool(nil),
		NewDeclineEDILoadTenderTool(nil),
		NewCancelEDILoadTenderTool(nil),
		NewExpireEDILoadTenderTool(nil),
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
		target, ok := tool.(agenttoolports.TargetedTool).Target(
			map[string]any{paramEDITransferID: id.String()},
		)
		require.True(t, ok, tool.Name())
		assert.Equal(t, agenttoolports.ToolTarget{Resource: permission.ResourceEDI, ID: id}, target)
	}
}

func TestAcceptEDILoadTender_PreviewsTheShipmentAndThe990ThenRunsOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(editransfer.TransferStatusPendingApproval),
		guard:    &agenttoolstest.WriteGuard{},
	}
	tool := NewAcceptEDILoadTenderTool(transfers).(*EdiTransferDecisionTool)
	args := tenderParams(transfers.transfer, nil)

	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		transfers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	status := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceEDI, status.Resource)
	assert.Equal(t, transfers.transfer.ID, status.EntityID)
	require.NotNil(t, status.Version)
	assert.Equal(t, int64(4), *status.Version)
	assert.Equal(t, "Processing", agenttoolstest.FieldByPath(t, status, "status").After)

	created := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, created.Resource)
	assert.Equal(t, agentpreview.PreviewOperationCreate, created.Operation)

	response := agenttoolstest.PreviewChange(t, preview, len(preview.Changes)-1)
	require.NotNil(t, response.Message)
	assert.Equal(t, agentpreview.MessageChannelEDI, response.Message.Channel)
	assert.Contains(t, response.Message.Body, "Accepted (A)")
	assert.Contains(t, response.Message.Body, "BOL-77")
	assert.Contains(t, preview.Summary, "Acme Foods")

	require.NoError(t, tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(args)))
	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)),
		ErrEDIDecisionNeedsAPerson,
	)
	assert.Nil(t, transfers.approved, "an agent never accepts a tender on its own")

	approved := agenttoolstest.ApprovedParams(args)
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, transfers.approved)
	assert.Equal(t, transfers.transfer.ID, transfers.approved.TransferID)
	assert.Equal(t, approved.OrganizationID, transfers.approved.TenantInfo.OrgID)
	assert.Equal(t, approved.Actor, transfers.actor)
}

func TestAcceptEDILoadTender_RefusesATenderWhoseMappingsAreIncomplete(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer:   inboundTender(editransfer.TransferStatusMappingRequired),
		unresolved: []editransfer.MappingResolution{{EntityType: edi.MappingEntityTypeCustomer}},
		guard:      &agenttoolstest.WriteGuard{},
	}
	tool := NewAcceptEDILoadTenderTool(transfers).(*EdiTransferDecisionTool)
	args := agenttoolstest.AgentParamsFor(tenderParams(transfers.transfer, nil))

	require.Error(t, tool.Validate(t.Context(), args))
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		transfers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), args)
		},
	)
	agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	assert.Empty(t, preview.Changes)
}

func TestDeclineEDILoadTender_CarriesTheReasonToThePartner(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(editransfer.TransferStatusPendingApproval),
		guard:    &agenttoolstest.WriteGuard{},
	}
	tool := NewDeclineEDILoadTenderTool(transfers).(*EdiTransferDecisionTool)

	missing := agenttoolstest.AgentParamsFor(tenderParams(transfers.transfer, nil))
	require.Error(t, tool.Validate(t.Context(), missing), "a decline needs a reason")

	args := tenderParams(transfers.transfer, map[string]any{
		changepreview.FieldReason: "No reefer capacity on that lane this week",
	})
	preview := agenttoolstest.PreviewWithoutWrites(
		t,
		transfers.guard,
		func() (*agentpreview.ToolPreview, error) {
			return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
		},
	)
	status := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, "Rejected", agenttoolstest.FieldByPath(t, status, "status").After)
	assert.Equal(t, "No reefer capacity on that lane this week",
		agenttoolstest.FieldByPath(t, status, "rejectionReason").After)
	response := agenttoolstest.PreviewChange(t, preview, 1)
	require.NotNil(t, response.Message)
	assert.Contains(t, response.Message.Body, "Declined (D)")
	assert.Contains(t, response.Message.Body, "No reefer capacity on that lane this week")
	assert.Equal(t, []string{ediservice.TransferDirectionInbound}, transfers.directions)

	require.ErrorIs(
		t,
		tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)),
		ErrEDIDecisionNeedsAPerson,
	)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(args)))
	require.NotNil(t, transfers.rejected)
	assert.Equal(t, "No reefer capacity on that lane this week", transfers.rejected.Reason)
}

func TestCancelAndExpireEDILoadTender_ReadTheSideTheServiceActsOn(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		build     func(ediTransferDecider) agenttoolports.AgentTool
		direction string
		status    string
		wrote     func(*fakeTransfers) bool
	}{
		"cancel": {
			build:     NewCancelEDILoadTenderTool,
			direction: ediservice.TransferDirectionOutbound,
			status:    "Canceled",
			wrote:     func(f *fakeTransfers) bool { return f.canceled != nil },
		},
		"expire": {
			build:     NewExpireEDILoadTenderTool,
			direction: ediservice.TransferDirectionAny,
			status:    "Expired",
			wrote:     func(f *fakeTransfers) bool { return f.expired != nil },
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			transfers := &fakeTransfers{
				transfer: inboundTender(editransfer.TransferStatusPendingApproval),
				guard:    &agenttoolstest.WriteGuard{},
			}
			transfers.transfer.InboundMessageID = pulid.Nil
			tool := tc.build(transfers).(*EdiTransferDecisionTool)
			args := tenderParams(transfers.transfer, nil)

			preview := agenttoolstest.PreviewWithoutWrites(
				t,
				transfers.guard,
				func() (*agentpreview.ToolPreview, error) {
					return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
				},
			)
			require.Len(t, preview.Changes, 1, "a tender between organizations sends no 990")
			assert.Equal(
				t,
				tc.status,
				agenttoolstest.FieldByPath(
					t,
					agenttoolstest.PreviewChange(t, preview, 0),
					"status",
				).After,
			)
			assert.Equal(t, []string{tc.direction}, transfers.directions)

			require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(args)))
			assert.True(t, tc.wrote(transfers))
		})
	}
}

func TestEDITenderDecisions_RefuseAFinalizedTransfer(t *testing.T) {
	t.Parallel()

	for _, build := range []func(ediTransferDecider) agenttoolports.AgentTool{
		NewAcceptEDILoadTenderTool,
		NewDeclineEDILoadTenderTool,
		NewCancelEDILoadTenderTool,
		NewExpireEDILoadTenderTool,
	} {
		transfers := &fakeTransfers{
			transfer: inboundTender(editransfer.TransferStatusApproved),
			guard:    &agenttoolstest.WriteGuard{},
		}
		tool := build(transfers).(*EdiTransferDecisionTool)
		args := agenttoolstest.AgentParamsFor(tenderParams(transfers.transfer, map[string]any{
			changepreview.FieldReason: "Duplicate of a tender already accepted",
		}))

		err := tool.Validate(t.Context(), args)
		require.Error(t, err, tool.Name())
		assert.Contains(t, err.Error(), "finalized or processing", tool.Name())

		preview := agenttoolstest.PreviewWithoutWrites(
			t,
			transfers.guard,
			func() (*agentpreview.ToolPreview, error) {
				return tool.Preview(t.Context(), args)
			},
		)
		agenttoolstest.RequireWarning(t, preview, agentpreview.PreviewWarningWouldFail)
	}
}

func TestEDITenderDecisions_RefuseAnActorFromAnotherTenant(t *testing.T) {
	t.Parallel()

	transfers := &fakeTransfers{
		transfer: inboundTender(editransfer.TransferStatusPendingApproval),
		guard:    &agenttoolstest.WriteGuard{},
	}
	tool := NewAcceptEDILoadTenderTool(transfers)
	params := agenttoolstest.ApprovedParams(tenderParams(transfers.transfer, nil))
	params.OrganizationID = pulid.MustNew("org_")

	require.ErrorIs(t, tool.Execute(t.Context(), params), toolguard.ErrTenantMismatch)
	assert.Nil(t, transfers.approved)
}
