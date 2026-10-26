package editools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeLoadTenders struct {
	plan      *ediservice.LoadTenderPlan
	err       error
	guard     *agenttoolstest.WriteGuard
	submitted *ediservice.SubmitLoadTenderRequest
	actor     *permissionports.RequestActor
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
	actor *permissionports.RequestActor,
) (*edi.EDITransfer, error) {
	if err := f.guard.Write(); err != nil {
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
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewSendEDILoadTenderTool(tenders).(*SendEDILoadTenderTool)
	partnerID := pulid.MustNew("edip_")
	args := map[string]any{
		common.FieldShipmentID: tenders.plan.SourceShipment.ID.String(),
		paramEDIPartnerID:      partnerID.String(),
	}

	policy := tool.Policy()
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	target, ok := tool.Target(args)
	require.True(t, ok)
	assert.Equal(t, permission.ResourceShipment, target.Resource)

	preview := agenttoolstest.PreviewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
	})
	sent := agenttoolstest.PreviewChange(t, preview, 0)
	require.NotNil(t, sent.Message)
	assert.Equal(t, agent.MessageChannelEDI, sent.Message.Channel)
	assert.Equal(t, []string{"Acme Brokerage"}, sent.Message.To)
	assert.Contains(t, sent.Message.Body, "Dallas DC")
	marked := agenttoolstest.PreviewChange(t, preview, 1)
	assert.Equal(t, permission.ResourceShipment, marked.Resource)
	require.NotNil(t, marked.Version)
	assert.Equal(t, int64(7), *marked.Version)
	assert.Equal(t, "Tendered", agenttoolstest.FieldByPath(t, marked, "tenderStatus").After)

	require.NoError(t, tool.Validate(t.Context(), agenttoolstest.AgentParamsFor(args)))
	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	require.NoError(t, tool.Execute(t.Context(), agenttoolstest.ApprovedParams(args)))
	require.NotNil(t, tenders.submitted)
	assert.Equal(t, partnerID, tenders.submitted.EDIPartnerID)
	assert.Equal(t, tenders.plan.SourceShipment.ID, tenders.submitted.SourceShipmentID)
}

func TestSendEDILoadTender_WarnsWhenTheReceiverMustMapItFirst(t *testing.T) {
	t.Parallel()

	tenders := &fakeLoadTenders{
		plan:  tenderPlan(edi.TransferStatusMappingRequired),
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewSendEDILoadTenderTool(tenders).(*SendEDILoadTenderTool)
	args := agenttoolstest.AgentParamsFor(map[string]any{
		common.FieldShipmentID: tenders.plan.SourceShipment.ID.String(),
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), args)
	})
	assert.Contains(t, preview.Summary, "map")

	tenders.err = errortypes.NewValidationError(
		"shipmentId", errortypes.ErrInvalidOperation,
		"Only New shipments without an active or accepted tender can be tendered.",
	)
	require.Error(t, tool.Validate(t.Context(), args))
	refused := agenttoolstest.PreviewWithoutWrites(t, tenders.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), args)
	})
	agenttoolstest.RequireWarning(t, refused, agent.PreviewWarningWouldFail)
}

type fakeDocuments struct {
	preview   *ediservice.EDIDocumentPreview
	err       error
	guard     *agenttoolstest.WriteGuard
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
	if err := f.guard.Write(); err != nil {
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
			Diagnostics: []edi.Diagnostic{{
				Severity: edi.ValidationSeverityWarning,
				Message:  "AT7 reason code defaulted",
			}},
		},
		guard: &agenttoolstest.WriteGuard{},
	}
	tool := NewSendEDIStatusUpdateTool(documents).(*SendEDIStatusUpdateTool)
	partnerID := pulid.MustNew("edip_")
	shipmentID := pulid.MustNew("shp_")
	args := map[string]any{
		paramEDIPartnerID:      partnerID.String(),
		common.FieldShipmentID: shipmentID.String(),
		"payload":              map[string]any{"transactionSet": "214"},
	}

	policy := tool.Policy()
	assert.Equal(t, permission.OpCreate, policy.Operation)
	assert.Equal(t, agent.TierPropose, policy.MaxTier)
	assert.NotContains(t, tool.ParamSchema()["properties"], "payload")

	preview := agenttoolstest.PreviewWithoutWrites(t, documents.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.AgentParamsFor(args))
	})
	sent := agenttoolstest.PreviewChange(t, preview, 0)
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

	require.ErrorIs(t, tool.Execute(t.Context(), agenttoolstest.AgentParamsFor(args)), ErrEDIDecisionNeedsAPerson)
	approved := agenttoolstest.ApprovedParams(args)
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, documents.generated)
	assert.Equal(t, shipmentID, documents.generated.ShipmentID)
	assert.Equal(t, approved.Actor.UserID, documents.generated.GeneratedByID)
	assert.Nil(t, documents.generated.Payload)
}

func TestSendEDIStatusUpdate_TakesExactlyOneSourceRecord(t *testing.T) {
	t.Parallel()

	tool := NewSendEDIStatusUpdateTool(&fakeDocuments{guard: &agenttoolstest.WriteGuard{}})
	validator := tool.(agentports.ToolValidator)
	partner := pulid.MustNew("edip_").String()

	none := agenttoolstest.AgentParamsFor(map[string]any{paramEDIPartnerID: partner})
	require.Error(t, validator.Validate(t.Context(), none))

	two := agenttoolstest.AgentParamsFor(map[string]any{
		paramEDIPartnerID:      partner,
		common.FieldShipmentID: pulid.MustNew("shp_").String(),
		ParamServiceFailureID:  pulid.MustNew("sf_").String(),
	})
	err := validator.Validate(t.Context(), two)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one")
}
