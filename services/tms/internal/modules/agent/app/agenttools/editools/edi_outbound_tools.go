package editools

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
)

const paramEDIPartnerID = "ediPartnerId"

const toolShipmentList = "shipment_list"

const ParamServiceFailureID = "serviceFailureId"

const ediPartnerIDGuidance = "from edi_list_partners or edi_get_partner. Never guess one."

const statusUpdateSourceHint = "shipmentId or serviceFailureId"

var errStatusUpdateSource = errors.New(
	"give exactly one of " + statusUpdateSourceHint + " for the status update to report",
)

type loadTenderSubmitter interface {
	PlanLoadTender(
		ctx context.Context,
		req *editenderservice.SubmitLoadTenderRequest,
	) (*editenderservice.LoadTenderPlan, error)
	SubmitLoadTender(
		ctx context.Context,
		req *editenderservice.SubmitLoadTenderRequest,
		actor *permissionports.RequestActor,
	) (*editransfer.EDITransfer, error)
}

type ediDocumentGenerator interface {
	PlanGenerateDocument(
		ctx context.Context,
		req *edioutbound.GenerateEDIDocumentRequest,
	) (*edioutbound.EDIDocumentPreview, error)
	GenerateDocument(
		ctx context.Context,
		req *edioutbound.GenerateEDIDocumentRequest,
	) (*edimessage.EDIMessage, error)
}

func ediCreatePolicy(name, rationale string) agenttoolports.ToolPolicy {
	policy := ediOutboundPolicy(name, rationale)
	policy.Operation = permission.OpCreate

	return policy
}

type SendEDILoadTenderTool struct {
	tenders loadTenderSubmitter
}

func NewSendEDILoadTenderTool(tenders loadTenderSubmitter) agenttoolports.AgentTool {
	return &SendEDILoadTenderTool{tenders: tenders}
}

func (t *SendEDILoadTenderTool) Name() string { return "edi_send_tender" }

func (t *SendEDILoadTenderTool) Prerequisites() []string {
	return []string{toolShipmentList, "edi_list_partners"}
}

func (t *SendEDILoadTenderTool) Recipe() []string {
	return []string{
		toolShipmentList,
		"edi_list_partners",
		"edi_send_tender",
		toolEDIListTransfers,
	}
}

func (t *SendEDILoadTenderTool) Description() string {
	return "Propose tendering a New shipment to another organization on Trenova over EDI, " +
		"which it accepts or declines on its side. The partner is the customer's internal EDI " +
		"partner unless you name one; the shipment is marked Tendered until they answer. A " +
		"shipment with a tender already out is refused. Carriers outside Trenova are " +
		"tendered with dispatch_create_tender instead. A person always decides."
}

func (t *SendEDILoadTenderTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		changepreview.FieldShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
			"The shipment to tender, from shipment_list or shipment_get. Never guess one."),
		paramEDIPartnerID: agenttoolschema.KindID("The internal EDI partner that stands for "+
			"the receiving organization, "+ediPartnerIDGuidance+" Leave it out to use the "+
			"customer's.", permission.KindEDIPartner),
	}, changepreview.FieldShipmentID)
}

func (t *SendEDILoadTenderTool) Policy() agenttoolports.ToolPolicy {
	return ediCreatePolicy(t.Name(),
		"Offers a load to another organization, which may accept it and haul it on the "+
			"terms tendered; a tender is withdrawn, never recalled.")
}

func (t *SendEDILoadTenderTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, changepreview.FieldShipmentID, permission.ResourceShipment)
}

func (t *SendEDILoadTenderTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*editenderservice.SubmitLoadTenderRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, changepreview.FieldShipmentID)
	if err != nil {
		return nil, err
	}
	partnerID, _, err := toolparams.OptionalPulid(params.Params, paramEDIPartnerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramEDIPartnerID, err)
	}

	return &editenderservice.SubmitLoadTenderRequest{
		TenantInfo:       toolparams.TenantFrom(*params),
		SourceShipmentID: shipmentID,
		EDIPartnerID:     partnerID,
	}, nil
}

func (t *SendEDILoadTenderTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.tenders.PlanLoadTender(ctx, req)

	return err
}

func (t *SendEDILoadTenderTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.tenders.SubmitLoadTender(ctx, req, params.Actor)

	return err
}

type SendEDIStatusUpdateTool struct {
	documents ediDocumentGenerator
}

func NewSendEDIStatusUpdateTool(documents ediDocumentGenerator) agenttoolports.AgentTool {
	return &SendEDIStatusUpdateTool{documents: documents}
}

func (t *SendEDIStatusUpdateTool) Name() string { return "edi_send_status_update" }

func (t *SendEDIStatusUpdateTool) Description() string {
	return "Propose sending a trading partner an EDI 214 shipment status built from a " +
		"shipment or a service failure. It reports the shipment's current status, or a late " +
		"or missed stop with its reason. Use it when a partner's automatic 214 did not go " +
		"or they ask for one again. The partner's document profile decides the format. An " +
		"invoice's 210 goes from the invoice, not here. A person always decides."
}

func (t *SendEDIStatusUpdateTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIPartnerID: agenttoolschema.KindID("The trading partner to send it to, "+
			ediPartnerIDGuidance, permission.KindEDIPartner),
		changepreview.FieldShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
			"Report this shipment's current status. The shipment, from shipment_list or "+
				"shipment_get."),
		ParamServiceFailureID: agenttoolschema.RecordIDText(
			permission.ResourceServiceFailure,
			"Report a late or missed stop with its reason, from exception_list_service_failures or "+
				"exception_get_service_failure.",
		),
	}, paramEDIPartnerID)
}

func (t *SendEDIStatusUpdateTool) Policy() agenttoolports.ToolPolicy {
	return ediCreatePolicy(t.Name(),
		"Sends a trading partner a shipment status they act on and pass to their own "+
			"customers; a document sent cannot be recalled.")
}

func (t *SendEDIStatusUpdateTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*edioutbound.GenerateEDIDocumentRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	partnerID, err := toolparams.RequirePulid(params.Params, paramEDIPartnerID)
	if err != nil {
		return nil, err
	}
	req := &edioutbound.GenerateEDIDocumentRequest{
		TenantInfo:     toolparams.TenantFrom(*params),
		EDIPartnerID:   partnerID,
		TransactionSet: edidocument.TransactionSet214,
		Direction:      edidocument.DocumentDirectionOutbound,
		GeneratedByID:  params.Actor.UserID,
	}

	sources := 0
	for key, target := range map[string]*pulid.ID{
		changepreview.FieldShipmentID: &req.ShipmentID,
		ParamServiceFailureID:         &req.ServiceFailureID,
	} {
		id, given, idErr := toolparams.OptionalPulid(params.Params, key)
		if idErr != nil {
			return nil, fmt.Errorf("parameter %q is not a valid id: %w", key, idErr)
		}
		if given {
			*target = id
			sources++
		}
	}
	if sources != 1 {
		return nil, errStatusUpdateSource
	}

	return req, nil
}

func (t *SendEDIStatusUpdateTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.documents.PlanGenerateDocument(ctx, req)

	return err
}

func (t *SendEDIStatusUpdateTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrEDIDecisionNeedsAPerson
	}

	req, err := t.request(&params)
	if err != nil {
		return err
	}

	if _, err = t.documents.GenerateDocument(ctx, req); err != nil {
		return fmt.Errorf("send the 214: %w", err)
	}

	return nil
}
