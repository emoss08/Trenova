package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLoadTenders struct {
	plan      *ediservice.LoadTenderPlan
	err       error
	guard     *writeGuard
	submitted *ediservice.SubmitLoadTenderRequest
	actor     *serviceports.RequestActor
}

func (f *fakeLoadTenders) PlanLoadTender(
	_ context.Context,
	req *ediservice.SubmitLoadTenderRequest,
) (*ediservice.LoadTenderPlan, error) {
	if f.err != nil {
		return nil, f.err
	}
	if req.SourceShipmentID != f.plan.SourceShipment.ID {
		return nil, errortypes.NewNotFoundError("Shipment not found")
	}

	return f.plan, nil
}

func (f *fakeLoadTenders) SubmitLoadTender(
	_ context.Context,
	req *ediservice.SubmitLoadTenderRequest,
	actor *serviceports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.submitted, f.actor = req, actor

	return &edi.EDITransfer{ID: pulid.MustNew("edilt_")}, nil
}

func tenderPlan(status edi.TransferStatus) *ediservice.LoadTenderPlan {
	return &ediservice.LoadTenderPlan{
		SourceShipment: &shipment.Shipment{
			ID:        pulid.MustNew("shp_"),
			ProNumber: "PRO-88",
			Status:    shipment.StatusNew,
			Version:   7,
		},
		SourcePartner: &edi.EDIPartner{ID: pulid.MustNew("edip_"), Name: "Acme Brokerage"},
		TargetPartner: &edi.EDIPartner{ID: pulid.MustNew("edip_"), Name: "Northwind Carrier"},
		Payload: edi.LoadTenderPayload{
			BOL:           "BOL-88",
			CustomerLabel: "Acme Foods",
			Moves: []edi.LoadTenderMove{{Stops: []edi.LoadTenderStop{
				{Type: "Pickup", LocationLabel: "Dallas DC"},
				{Type: "Delivery", LocationLabel: "Tulsa Store"},
			}}},
		},
		Mapping: &ediservice.MappingPreview{},
		Status:  status,
	}
}

func TestSendEDILoadTender_PreviewsTheTenderAndTheShipmentThenSendsOnlyWhenApproved(t *testing.T) {
	t.Parallel()

	tenders := &fakeLoadTenders{
		plan:  tenderPlan(edi.TransferStatusPendingApproval),
		guard: &writeGuard{},
	}
	tool := newSendEDILoadTenderTool(tenders).(*sendEDILoadTenderTool)
	partnerID := pulid.MustNew("edip_")
	args := map[string]any{
		fieldShipmentID:   tenders.plan.SourceShipment.ID.String(),
		paramEDIPartnerID: partnerID.String(),
	}

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	target, ok := tool.Target(args)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, target.Resource)

	preview := previewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	sent := previewChange(t, preview, 0)
	require.NotNil(t, sent.Message)
	assert.Equal(t, agent.MessageChannelEDI, sent.Message.Channel)
	assert.Equal(t, []string{"Acme Brokerage"}, sent.Message.To)
	assert.Contains(t, sent.Message.Body, "Dallas DC")
	marked := previewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, marked.Resource)
	require.NotNil(t, marked.Version)
	assert.Equal(t, int64(7), *marked.Version)
	assert.Equal(t, "Tendered", fieldByPath(t, marked, "tenderStatus").After)

	require.NoError(t, tool.Validate(t.Context(), agentParamsFor(args)))
	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), approvedParams(args)))
	require.NotNil(t, tenders.submitted)
	assert.Equal(t, partnerID, tenders.submitted.EDIPartnerID)
	assert.Equal(t, tenders.plan.SourceShipment.ID, tenders.submitted.SourceShipmentID)
}

func TestSendEDILoadTender_WarnsWhenTheReceiverMustMapItFirst(t *testing.T) {
	t.Parallel()

	tenders := &fakeLoadTenders{
		plan:  tenderPlan(edi.TransferStatusMappingRequired),
		guard: &writeGuard{},
	}
	tool := newSendEDILoadTenderTool(tenders).(*sendEDILoadTenderTool)
	args := agentParamsFor(map[string]any{
		fieldShipmentID: tenders.plan.SourceShipment.ID.String(),
	})

	preview := previewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), args)
	})
	assert.Contains(t, preview.Summary, "map")

	tenders.err = errortypes.NewValidationError(
		"shipmentId", errortypes.ErrInvalidOperation,
		"Only New shipments without an active or accepted tender can be tendered.",
	)
	require.Error(t, tool.Validate(t.Context(), args))
	refused := previewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), args)
	})
	requireWarning(t, refused, agent.PreviewWarningWouldFail)
}

type fakeDocuments struct {
	preview   *ediservice.EDIDocumentPreview
	err       error
	guard     *writeGuard
	planned   *ediservice.GenerateEDIDocumentRequest
	generated *ediservice.GenerateEDIDocumentRequest
}

func (f *fakeDocuments) PlanGenerateDocument(
	_ context.Context,
	req *ediservice.GenerateEDIDocumentRequest,
) (*ediservice.EDIDocumentPreview, error) {
	f.planned = req

	return f.preview, f.err
}

func (f *fakeDocuments) GenerateDocument(
	_ context.Context,
	req *ediservice.GenerateEDIDocumentRequest,
) (*edi.EDIMessage, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.generated = req

	return &edi.EDIMessage{ID: pulid.MustNew("edimsg_")}, nil
}

func TestSendEDIStatusUpdate_PreviewsTheRenderedStatusAndNeverTakesAPayload(t *testing.T) {
	t.Parallel()

	documents := &fakeDocuments{
		preview: &ediservice.EDIDocumentPreview{
			RawX12:       "ISA*00*~ST*214*0001~B10*PRO-88*BOL-88~",
			SegmentCount: 3,
			Profile:      &edi.EDIPartnerDocumentProfile{Name: "Acme 214"},
			Diagnostics: []edix12.Diagnostic{{
				Severity: edi.ValidationSeverityWarning,
				Message:  "AT7 reason code defaulted",
			}},
		},
		guard: &writeGuard{},
	}
	tool := newSendEDIStatusUpdateTool(documents).(*sendEDIStatusUpdateTool)
	partnerID := pulid.MustNew("edip_")
	shipmentID := pulid.MustNew("shp_")
	args := map[string]any{
		paramEDIPartnerID: partnerID.String(),
		fieldShipmentID:   shipmentID.String(),
		"payload":         map[string]any{"transactionSet": "214"},
	}

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.NotContains(t, tool.ParamSchema()["properties"], "payload")

	preview := previewWithoutWrites(t, documents.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agentParamsFor(args))
	})
	sent := previewChange(t, preview, 0)
	require.NotNil(t, sent.Message)
	assert.Contains(t, sent.Message.Body, "B10*PRO-88")
	assert.Contains(t, preview.Summary, "AT7 reason code defaulted")
	require.NotNil(t, documents.planned)
	assert.Nil(
		t,
		documents.planned.Payload,
		"the document is built from the record, never the model",
	)
	assert.Equal(t, edi.TransactionSet214, documents.planned.TransactionSet)
	assert.Equal(t, edi.DocumentDirectionOutbound, documents.planned.Direction)

	require.ErrorIs(t, tool.Execute(t.Context(), agentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	approved := approvedParams(args)
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, documents.generated)
	assert.Equal(t, shipmentID, documents.generated.ShipmentID)
	assert.Equal(t, approved.Actor.UserID, documents.generated.GeneratedByID)
	assert.Nil(t, documents.generated.Payload)
}

func TestSendEDIStatusUpdate_TakesExactlyOneSourceRecord(t *testing.T) {
	t.Parallel()

	tool := newSendEDIStatusUpdateTool(&fakeDocuments{guard: &writeGuard{}})
	validator := tool.(serviceports.ToolValidator)
	partner := pulid.MustNew("edip_").String()

	none := agentParamsFor(map[string]any{paramEDIPartnerID: partner})
	require.Error(t, validator.Validate(t.Context(), none))

	two := agentParamsFor(map[string]any{
		paramEDIPartnerID:     partner,
		fieldShipmentID:       pulid.MustNew("shp_").String(),
		paramServiceFailureID: pulid.MustNew("sf_").String(),
	})
	err := validator.Validate(t.Context(), two)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}
