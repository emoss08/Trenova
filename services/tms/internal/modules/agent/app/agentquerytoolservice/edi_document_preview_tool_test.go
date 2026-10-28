package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rendered214 = "ISA*00*          *00*          *ZZ*TRENOVA        *ZZ*ACMERETAIL     " +
	"*261014*0930*U*00401*000000000*0*T*>~GS*QM*TRENOVA*ACMERETAIL*20261014*0930*0*X*004010~" +
	"ST*214*0000~B10**BOL-88*TRNV~LX*1~AT7*X3****20261014*0915~SE*5*0000~GE*1*0~" +
	"IEA*1*000000000~"

type fakeDocumentPreviewer struct {
	outbound     *ediservice.OutboundDocumentPreview
	response     *ediservice.TenderResponsePreview
	outboundReq  *ediservice.GenerateEDIDocumentRequest
	responseReq  *ediservice.PreviewTenderResponseRequest
	previewCalls int
}

func (f *fakeDocumentPreviewer) PreviewOutboundDocument(
	_ context.Context,
	req *ediservice.GenerateEDIDocumentRequest,
) (*ediservice.OutboundDocumentPreview, error) {
	f.previewCalls++
	f.outboundReq = req

	return f.outbound, nil
}

func (f *fakeDocumentPreviewer) PreviewTenderResponse(
	_ context.Context,
	req *ediservice.PreviewTenderResponseRequest,
) (*ediservice.TenderResponsePreview, error) {
	f.previewCalls++
	f.responseReq = req

	return f.response, nil
}

func readers(resources ...permission.Resource) *fakePermissions {
	readable := make(map[string]*permissionports.ResourcePermissionDetail, len(resources))
	for _, resource := range resources {
		readable[resource.String()] = &permissionports.ResourcePermissionDetail{
			Resource:       resource.String(),
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityRestricted,
		}
	}

	return &fakePermissions{readable: readable}
}

func TestPreviewEDIDocument_RendersA214ThroughTheSendPathWithItsFindings(t *testing.T) {
	t.Parallel()

	fake := &fakeDocumentPreviewer{outbound: &ediservice.OutboundDocumentPreview{
		Document: &ediservice.EDIDocumentPreview{
			RawX12:       rendered214,
			SegmentCount: 5,
			X12Version:   "004010",
			Diagnostics: []edidocument.Diagnostic{{
				Severity:        edidocument.ValidationSeverityError,
				Code:            "required",
				SegmentID:       "B10",
				ElementPosition: 1,
				Message:         "Reference Identification is required",
				SuggestedFix:    "Give the shipment a PRO or BOL",
			}},
			PartnerName: "Acme Retail",
			Map:         "statusOut",
		},
		Blocking: true,
	}}
	partnerID := pulid.MustNew("edip_")
	shipmentID := pulid.MustNew("shp_")
	tool := newPreviewEDIDocumentTool(
		fake,
		readers(permission.ResourceEDI, permission.ResourceShipment),
	)

	params := chatParams(map[string]any{
		paramTransactionSet: "214",
		paramPartnerID:      partnerID.String(),
		paramShipmentID:     shipmentID.String(),
	}, "")
	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	req := fake.outboundReq
	require.NotNil(t, req)
	assert.Equal(t, partnerID, req.EDIPartnerID)
	assert.Equal(t, shipmentID, req.ShipmentID)
	assert.Equal(t, edidocument.TransactionSet214, req.TransactionSet)
	assert.Equal(t, edidocument.DocumentDirectionOutbound, req.Direction,
		"the request edi_send_status_update makes")
	assert.Equal(t, params.OrganizationID, req.TenantInfo.OrgID)

	view := result.(*ediDocumentPreviewView)
	assert.Equal(t, "EDI 214 to Acme Retail", view.Title)
	assert.False(t, view.WouldSend, "an error finding would refuse it as it stands")
	assert.Equal(t, "statusOut", view.Map)
	assert.Equal(t, rendered214, view.Raw)
	require.Len(t, view.Segments, 9, "ISA through IEA")
	assert.Equal(t, "ST*214*0000", view.Segments[2])
	require.Len(t, view.Findings, 1)
	assert.Equal(t, ediDiagnosticRow{
		Severity:     "Error",
		Code:         "required",
		Segment:      "B10",
		Element:      1,
		Message:      "Reference Identification is required",
		SuggestedFix: "Give the shipment a PRO or BOL",
	}, view.Findings[0])
	assert.Contains(t, view.Note, "control numbers")
	assert.Empty(t, view.TaintedRecords(), "a document built from the shipment is not partner text")
}

func TestPreviewEDIDocument_AnswersATenderWithThe990ItWouldSend(t *testing.T) {
	t.Parallel()

	transfer := &editransfer.EDITransfer{
		ID:     pulid.MustNew("edilt_"),
		Status: editransfer.TransferStatusPendingApproval,
	}
	fake := &fakeDocumentPreviewer{response: &ediservice.TenderResponsePreview{
		OutboundDocumentPreview: ediservice.OutboundDocumentPreview{
			Document: &ediservice.EDIDocumentPreview{
				RawX12:      "ST*990*0000~B1*BOLT*BOL-5*20261014*D~SE*3*0000~",
				PartnerName: "Bolt Freight",
			},
		},
		Transfer: transfer,
		Choice:   ediservice.TenderResponseDecline,
	}}
	partnerID := pulid.MustNew("edip_")
	tool := newPreviewEDIDocumentTool(fake, readers(permission.ResourceEDI))

	result, err := tool.Query(t.Context(), chatParams(map[string]any{
		paramTransactionSet: "990",
		paramPartnerID:      partnerID.String(),
		paramEDITransferID:  transfer.ID.String(),
		paramTenderResponse: "Decline",
		paramDeclineReason:  "No trucks Friday",
	}, ""))
	require.NoError(t, err)

	require.NotNil(t, fake.responseReq)
	assert.Equal(t, transfer.ID, fake.responseReq.TransferID)
	assert.Equal(t, partnerID, fake.responseReq.PartnerID)
	assert.Equal(t, ediservice.TenderResponseDecline, fake.responseReq.Choice)
	assert.Equal(t, "No trucks Friday", fake.responseReq.Reason)

	view := result.(*ediDocumentPreviewView)
	assert.Equal(t, "Decline", view.Response)
	assert.True(t, view.WouldSend)
	assert.Equal(t,
		[]agent.RecordRef{{EntityType: ediTransferEntity, ID: transfer.ID.String()}},
		view.TaintedRecords(), "a 990 repeats the partner's tender")
}

func TestPreviewEDIDocument_WithholdsTheX12BelowRestrictedButKeepsTheFindings(t *testing.T) {
	t.Parallel()

	fake := &fakeDocumentPreviewer{outbound: &ediservice.OutboundDocumentPreview{
		Document: &ediservice.EDIDocumentPreview{
			RawX12:      rendered214,
			PartnerName: "Acme Retail",
			Diagnostics: []edidocument.Diagnostic{
				{Severity: edidocument.ValidationSeverityWarning, Message: "N1 missing"},
			},
		},
	}}
	tool := newPreviewEDIDocumentTool(fake, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		paramTransactionSet: "214",
		paramPartnerID:      pulid.MustNew("edip_").String(),
		paramShipmentID:     pulid.MustNew("shp_").String(),
	}, permission.SensitivityInternal))
	require.NoError(t, err)

	view := result.(*ediDocumentPreviewView)
	assert.Empty(t, view.Raw)
	assert.Empty(t, view.Segments)
	assert.Equal(t, []string{"rawX12"}, view.Withheld)
	require.Len(t, view.Findings, 1)
	assert.True(t, view.WouldSend)
	assert.Equal(t, "Raw EDI is Restricted and this agent reads at Internal, so the X12 is held "+
		"back. Tell the person that someone whose role reaches Restricted can set this agent's "+
		"Data access to Restricted in AI control. The findings and whether it would go stand. "+
		"Nothing was sent or saved.", view.Note)
}

func TestPreviewEDIDocument_SaysWhoseAccessHeldTheX12Back(t *testing.T) {
	t.Parallel()

	preview := func(
		t *testing.T,
		permissions *fakePermissions,
		agentAccess permission.FieldSensitivity,
	) *ediDocumentPreviewView {
		t.Helper()

		permissions.readable[permission.ResourceShipment.String()] = &permissionports.ResourcePermissionDetail{
			Resource:       permission.ResourceShipment.String(),
			Operations:     []permission.Operation{permission.OpRead},
			MaxSensitivity: permission.SensitivityRestricted,
		}

		fake := &fakeDocumentPreviewer{outbound: &ediservice.OutboundDocumentPreview{
			Document: &ediservice.EDIDocumentPreview{
				RawX12:      rendered214,
				PartnerName: "Acme Retail",
			},
		}}
		result, err := newPreviewEDIDocumentTool(
			fake,
			permissions,
		).Query(t.Context(), chatParams(map[string]any{
			paramTransactionSet: "214",
			paramPartnerID:      pulid.MustNew("edip_").String(),
			paramShipmentID:     pulid.MustNew("shp_").String(),
		}, agentAccess))
		require.NoError(t, err)

		return result.(*ediDocumentPreviewView)
	}

	byAgent := preview(t,
		personReaching(permission.ResourceEDI, permission.SensitivityRestricted),
		permission.SensitivityInternal)
	assert.Empty(t, byAgent.Segments)
	assert.Contains(t, byAgent.Note, "this agent reads at Internal")

	byRole := preview(t,
		personReaching(permission.ResourceEDI, permission.SensitivityInternal),
		permission.SensitivityRestricted)
	assert.Empty(t, byRole.Segments)
	assert.Contains(t, byRole.Note, "the person's role does not reach it")
	assert.NotContains(t, byRole.Note, "AI control", "raising the agent would not help")

	shown := preview(t,
		personReaching(permission.ResourceEDI, permission.SensitivityRestricted),
		permission.SensitivityRestricted)
	assert.NotEmpty(t, shown.Segments)
	assert.Empty(t, shown.Withheld)
}

func TestPreviewEDIDocument_RefusesWhatItCannotPreview(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_").String()
	shipmentID := pulid.MustNew("shp_").String()
	transferID := pulid.MustNew("edilt_").String()
	cases := map[string]struct {
		params      map[string]any
		permissions *fakePermissions
		contains    string
	}{
		"a 214 with no shipment": {
			params:      map[string]any{paramTransactionSet: "214", paramPartnerID: partnerID},
			permissions: readers(permission.ResourceEDI, permission.ResourceShipment),
			contains:    paramShipmentID,
		},
		"a 990 named by a shipment": {
			params: map[string]any{
				paramTransactionSet: "990", paramPartnerID: partnerID, paramShipmentID: shipmentID,
			},
			permissions: readers(permission.ResourceEDI, permission.ResourceShipment),
			contains:    paramEDITransferID,
		},
		"a 204 named by both": {
			params: map[string]any{
				paramTransactionSet: "204", paramPartnerID: partnerID,
				paramShipmentID: shipmentID, paramEDITransferID: transferID,
			},
			permissions: readers(permission.ResourceEDI, permission.ResourceShipment),
			contains:    paramEDITransferID,
		},
		"an answer on a 214": {
			params: map[string]any{
				paramTransactionSet: "214", paramPartnerID: partnerID, paramShipmentID: shipmentID,
				paramTenderResponse: "Accept",
			},
			permissions: readers(permission.ResourceEDI, permission.ResourceShipment),
			contains:    paramTenderResponse,
		},
		"a shipment the person may not read": {
			params: map[string]any{
				paramTransactionSet: "214", paramPartnerID: partnerID, paramShipmentID: shipmentID,
			},
			permissions: readers(permission.ResourceEDI),
			contains:    "may not read",
		},
		"a set it does not preview": {
			params: map[string]any{
				paramTransactionSet: "210", paramPartnerID: partnerID, paramShipmentID: shipmentID,
			},
			permissions: readers(permission.ResourceEDI, permission.ResourceShipment),
			contains:    paramTransactionSet,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fake := &fakeDocumentPreviewer{}
			_, err := newPreviewEDIDocumentTool(fake, tc.permissions).
				Query(t.Context(), chatParams(tc.params, ""))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.contains)
			assert.Zero(t, fake.previewCalls, "nothing is rendered for a refused call")
		})
	}
}

func TestPreviewEDIDocument_IsAReadOnEDI(t *testing.T) {
	t.Parallel()

	policy := newPreviewEDIDocumentTool(&fakeDocumentPreviewer{}, &fakePermissions{}).Policy()
	assert.Equal(t, agent.ToolKindQuery, policy.Kind)
	assert.Equal(t, permission.ResourceEDI, policy.Resource)
	assert.Equal(t, permission.OpRead, policy.Operation)
	assert.Equal(t, agent.ExternalReadMarked, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceEDI, policy.Source)
	assert.False(t, policy.CapsLaterWrites)
}
