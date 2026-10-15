package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12"
	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramTenderResponse = "response"
	paramDeclineReason  = "reason"

	maxPreviewSegments     = 200
	maxDeclineReasonLength = 500

	rawX12Field = "rawX12"

	previewDocumentNote = "Rendered as a send would render it now, with provisional control " +
		"numbers (zeros); the real ones are assigned when it is sent. Nothing was sent or saved."
	previewWithheldNote = " The findings and whether it would go stand. Nothing was sent or saved."
)

var (
	previewTransactionSets = agenttoolschema.Source("edi.previewTransactionSet", []string{
		string(edi.TransactionSet204),
		string(edi.TransactionSet214),
		string(edi.TransactionSet990),
	})
	tenderResponses = agenttoolschema.Source("edi.tenderResponseChoice", []string{
		string(ediservice.TenderResponseAccept),
		string(ediservice.TenderResponseDecline),
	})
)

type ediDocumentPreviewer interface {
	PreviewOutboundDocument(
		ctx context.Context,
		req *ediservice.GenerateEDIDocumentRequest,
	) (*ediservice.OutboundDocumentPreview, error)
	PreviewTenderResponse(
		ctx context.Context,
		req *ediservice.PreviewTenderResponseRequest,
	) (*ediservice.TenderResponsePreview, error)
}

func providePreviewEDIDocumentTool(
	ediSvc *ediservice.Service,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newPreviewEDIDocumentTool(ediSvc, permissions)
}

type previewEDIDocumentTool struct {
	documents ediDocumentPreviewer
	access    fieldAccess
}

func newPreviewEDIDocumentTool(
	documents ediDocumentPreviewer,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &previewEDIDocumentTool{documents: documents, access: newFieldAccess(permissions)}
}

func (t *previewEDIDocumentTool) Name() string { return "preview_edi_document" }

func (t *previewEDIDocumentTool) SearchTerms() []string {
	return []string{
		"what would we send", "before it goes", "render 214", "render 204", "preview the 990",
		"990 look like", "990 we would send", "edi document preview", "check before sending",
	}
}

func (t *previewEDIDocumentTool) Description() string {
	return "Render the X12 a send would produce now, without sending or saving it: a 204 or " +
		"214 for a shipment, or the 990 answering a partner's tender. It returns each " +
		"segment, the partner profile's validation findings with the segment and element " +
		"they name, and whether the profile would let it go. For a 990 on an unanswered " +
		"tender, say whether to preview accepting or declining it."
}

func (t *previewEDIDocumentTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramTransactionSet: enumParam("Which document: 204 load tender, 214 shipment status "+
			"or 990 tender response.", previewTransactionSets),
		paramPartnerID: agenttoolschema.KindID("The trading partner it would go to, by id from "+
			"list_edi_partners or get_edi_partner.", permission.KindEDIPartner),
		paramShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
			"For a 204 or 214: the shipment, from search_shipments or get_shipment."),
		paramEDITransferID: agenttoolschema.KindID("For a 990: the tender it answers, from "+
			"list_edi_transfers.", permission.KindEDITransfer),
		paramTenderResponse: enumParam("For a 990 on a tender not yet answered: preview "+
			"accepting or declining it.", tenderResponses),
		paramDeclineReason: stringParam(fmt.Sprintf(
			"For a declining 990: the reason, at most %d characters.", maxDeclineReasonLength,
		)),
	}, paramTransactionSet, paramPartnerID)
}

func (t *previewEDIDocumentTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceEDI,
		reads:    agent.ExternalReadMarked,
		source:   agent.TaintSourceEDI,
		rationale: "Renders the X12 a send would produce from records already in Trenova, " +
			"with the partner profile's findings; nothing is saved and nothing is sent.",
	})
}

type ediDocumentPreviewView struct {
	Title          string             `json:"title"`
	TransactionSet string             `json:"transactionSet"`
	Partner        string             `json:"partner,omitempty"`
	Profile        string             `json:"profile,omitempty"`
	ValidationMode string             `json:"validationMode,omitempty"`
	X12Version     string             `json:"x12Version,omitempty"`
	Response       string             `json:"response,omitempty"`
	WouldSend      bool               `json:"wouldSend"`
	SegmentCount   int64              `json:"segmentCount"`
	Segments       []string           `json:"segments"`
	SegmentsCut    int                `json:"segmentsNotShown,omitempty"`
	FindingCount   int                `json:"findingCount"`
	Findings       []ediDiagnosticRow `json:"findings"`
	rawX12Excerpt
	Note string `json:"note"`

	outside []agent.RecordRef
}

func (v *ediDocumentPreviewView) TaintedRecords() []agent.RecordRef { return v.outside }

type documentPreviewCall struct {
	set        edi.TransactionSet
	partnerID  pulid.ID
	shipmentID pulid.ID
	transferID pulid.ID
	response   ediservice.TenderResponseChoice
	reason     string
}

func (t *previewEDIDocumentTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	call, err := readDocumentPreviewCall(params.Params)
	if err != nil {
		return nil, err
	}
	if err = t.mayReadSource(ctx, params, call); err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	if call.set == edi.TransactionSet990 {
		answered, pErr := t.documents.PreviewTenderResponse(
			ctx,
			&ediservice.PreviewTenderResponseRequest{
				TenantInfo: tenant,
				PartnerID:  call.partnerID,
				TransferID: call.transferID,
				Choice:     call.response,
				Reason:     call.reason,
			},
		)
		if pErr != nil {
			return nil, pErr
		}
		view := documentPreviewView(call.set, &answered.OutboundDocumentPreview)
		view.Response = string(answered.Choice)
		view.outside = []agent.RecordRef{
			{EntityType: ediTransferEntity, ID: call.transferID.String()},
		}

		return t.gated(ctx, params, view), nil
	}

	preview, err := t.documents.PreviewOutboundDocument(ctx, &ediservice.GenerateEDIDocumentRequest{
		TenantInfo:     tenant,
		EDIPartnerID:   call.partnerID,
		ShipmentID:     call.shipmentID,
		TransactionSet: call.set,
		Direction:      edi.DocumentDirectionOutbound,
		GeneratedByID:  params.Actor.UserID,
	})
	if err != nil {
		return nil, err
	}

	return t.gated(ctx, params, documentPreviewView(call.set, preview)), nil
}

func (t *previewEDIDocumentTool) gated(
	ctx context.Context,
	params *agentports.QueryToolParams,
	view *ediDocumentPreviewView,
) *ediDocumentPreviewView {
	gate := t.access.gate(ctx, params, permission.ResourceEDI)
	if gate.show(rawX12Field, rawX12Field) {
		return view
	}

	view.Raw = ""
	view.RawTruncated = false
	view.Segments = []string{}
	view.SegmentsCut = 0
	view.Withheld = gate.Withheld()
	view.Note = rawEDIWithheldNote(gate, rawX12Field) + previewWithheldNote

	return view
}

func readDocumentPreviewCall(params map[string]any) (*documentPreviewCall, error) {
	set, err := validEnum(params, paramTransactionSet, previewTransactionSets)
	if err != nil {
		return nil, err
	}
	if set == "" {
		return nil, fmt.Errorf("parameter %q is required: 204, 214 or 990", paramTransactionSet)
	}
	call := &documentPreviewCall{set: edi.TransactionSet(set)}
	if call.partnerID, err = requirePulid(params, paramPartnerID); err != nil {
		return nil, err
	}
	if call.shipmentID, err = optionalID(params, paramShipmentID); err != nil {
		return nil, err
	}
	if call.transferID, err = optionalID(params, paramEDITransferID); err != nil {
		return nil, err
	}
	response, err := validEnum(params, paramTenderResponse, tenderResponses)
	if err != nil {
		return nil, err
	}
	call.response = ediservice.TenderResponseChoice(response)
	call.reason = strings.TrimSpace(optionalString(params, paramDeclineReason))

	if call.set == edi.TransactionSet990 {
		err = call.check990()
	} else {
		err = call.checkShipmentDocument()
	}
	if err != nil {
		return nil, err
	}

	return call, nil
}

func (c *documentPreviewCall) check990() error {
	switch {
	case c.transferID.IsNil():
		return fmt.Errorf("a 990 answers a tender: give %q from list_edi_transfers",
			paramEDITransferID)
	case c.shipmentID.IsNotNil():
		return fmt.Errorf("a 990 is named by %q, not %q; leave the shipment out",
			paramEDITransferID, paramShipmentID)
	case c.reason != "" && c.response != ediservice.TenderResponseDecline:
		return fmt.Errorf("parameter %q goes only with %q Decline", paramDeclineReason,
			paramTenderResponse)
	case len(c.reason) > maxDeclineReasonLength:
		return fmt.Errorf("parameter %q is longer than %d characters", paramDeclineReason,
			maxDeclineReasonLength)
	}

	return nil
}

func (c *documentPreviewCall) checkShipmentDocument() error {
	switch {
	case c.shipmentID.IsNil():
		return fmt.Errorf("a %s is built from a shipment: give %q", c.set, paramShipmentID)
	case c.transferID.IsNotNil():
		return fmt.Errorf("a %s is named by %q alone; %q is for a 990", c.set, paramShipmentID,
			paramEDITransferID)
	case c.response != "" || c.reason != "":
		return fmt.Errorf("parameters %q and %q are for a 990 only", paramTenderResponse,
			paramDeclineReason)
	}

	return nil
}

func (t *previewEDIDocumentTool) mayReadSource(
	ctx context.Context,
	params *agentports.QueryToolParams,
	call *documentPreviewCall,
) error {
	if call.set == edi.TransactionSet990 {
		if !t.access.mayReadRecord(ctx, params, permission.ResourceEDI, call.transferID.String()) {
			return errors.New("the person you are working for may not read that tender, so " +
				"its 990 was not rendered")
		}

		return nil
	}
	if !t.access.mayReadRecord(ctx, params, permission.ResourceShipment, call.shipmentID.String()) {
		return errors.New("the person you are working for may not read that shipment, so " +
			"no document was rendered from it")
	}

	return nil
}

func documentPreviewView(
	set edi.TransactionSet,
	preview *ediservice.OutboundDocumentPreview,
) *ediDocumentPreviewView {
	document := preview.Document
	view := &ediDocumentPreviewView{
		Title:          "EDI " + string(set),
		TransactionSet: string(set),
		X12Version:     document.X12Version,
		WouldSend:      !preview.Blocking,
		SegmentCount:   document.SegmentCount,
		FindingCount:   len(document.Diagnostics),
		Findings:       previewFindings(document.Diagnostics),
		Note:           previewDocumentNote,
	}
	var envelope *edi.X12EnvelopeSettings
	if profile := document.Profile; profile != nil {
		envelope = &profile.Envelope
		view.Profile = profile.Name
		view.ValidationMode = string(profile.ValidationMode)
		if profile.Partner != nil {
			view.Partner = profile.Partner.Name
		}
	}
	if partner := stringutils.FirstNonEmpty(view.Partner, view.Profile); partner != "" {
		view.Title += " to " + partner
	}

	inspection := edix12inspect.InspectX12(&edix12inspect.InspectX12Request{
		RawX12:         document.RawX12,
		TransactionSet: set,
		X12Version:     document.X12Version,
		Envelope:       envelope,
	})
	view.Segments = make([]string, 0, min(len(inspection.Segments), maxPreviewSegments))
	for idx := range inspection.Segments {
		if len(view.Segments) == maxPreviewSegments {
			view.SegmentsCut = len(inspection.Segments) - maxPreviewSegments
			break
		}
		view.Segments = append(view.Segments, inspection.Segments[idx].Raw)
	}
	view.Raw = stringutils.TruncateBytes(document.RawX12, maxRawX12Bytes)
	view.RawTruncated = len(view.Raw) < len(document.RawX12)

	return view
}

func previewFindings(diagnostics []edix12.Diagnostic) []ediDiagnosticRow {
	rows := make([]ediDiagnosticRow, 0, min(len(diagnostics), maxMessageDiagnostics))
	for idx := range diagnostics {
		if len(rows) == maxMessageDiagnostics {
			break
		}
		diagnostic := &diagnostics[idx]
		rows = append(rows, ediDiagnosticRow{
			Severity:     string(diagnostic.Severity),
			Code:         diagnostic.Code,
			Segment:      diagnostic.SegmentID,
			Element:      diagnostic.ElementPosition,
			Path:         diagnostic.Path,
			Message:      diagnostic.Message,
			SuggestedFix: diagnostic.SuggestedFix,
		})
	}

	return rows
}
