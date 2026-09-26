package agenttoolservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramEDIPartnerID      = "ediPartnerId"
	paramServiceFailureID  = "serviceFailureId"
	ediPartnerIDGuidance   = "from list_edi_partners or get_edi_partner. Never guess one."
	statusUpdateSourceHint = "shipmentId or serviceFailureId"
)

var errStatusUpdateSource = errors.New(
	"give exactly one of " + statusUpdateSourceHint + " for the status update to report",
)

type loadTenderSubmitter interface {
	PlanLoadTender(
		ctx context.Context,
		req *ediservice.SubmitLoadTenderRequest,
	) (*ediservice.LoadTenderPlan, error)
	SubmitLoadTender(
		ctx context.Context,
		req *ediservice.SubmitLoadTenderRequest,
		actor *serviceports.RequestActor,
	) (*edi.EDITransfer, error)
}

type ediDocumentGenerator interface {
	PlanGenerateDocument(
		ctx context.Context,
		req *ediservice.GenerateEDIDocumentRequest,
	) (*ediservice.EDIDocumentPreview, error)
	GenerateDocument(
		ctx context.Context,
		req *ediservice.GenerateEDIDocumentRequest,
	) (*edi.EDIMessage, error)
}

func ediCreatePolicy(name, rationale string) serviceports.ToolPolicy {
	policy := ediOutboundPolicy(name, rationale)
	policy.Operation = permission.OpCreate

	return policy
}

type sendEDILoadTenderTool struct {
	tenders loadTenderSubmitter
}

var (
	_ serviceports.ToolPreviewer = (*sendEDILoadTenderTool)(nil)
	_ serviceports.ToolValidator = (*sendEDILoadTenderTool)(nil)
	_ serviceports.TargetedTool  = (*sendEDILoadTenderTool)(nil)
)

func newSendEDILoadTenderTool(tenders loadTenderSubmitter) serviceports.AgentTool {
	return &sendEDILoadTenderTool{tenders: tenders}
}

func (t *sendEDILoadTenderTool) Name() string { return "send_edi_tender" }

func (t *sendEDILoadTenderTool) Description() string {
	return "Propose tendering a New shipment to another organization on Trenova over EDI, " +
		"which it accepts or declines on its side. The partner is the customer's internal EDI " +
		"partner unless you name one; the shipment is marked Tendered until they answer. A " +
		"shipment with a tender already out is refused. Carriers outside Trenova are " +
		"tendered with tender_move_to_carriers instead. A person always decides."
}

func (t *sendEDILoadTenderTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		fieldShipmentID: jsonschemautils.Text("The shipment to tender, from search_shipments " +
			"or get_shipment. Never guess one."),
		paramEDIPartnerID: jsonschemautils.Text("The internal EDI partner that stands for the " +
			"receiving organization, " + ediPartnerIDGuidance + " Leave it out to use the " +
			"customer's."),
	}, fieldShipmentID)
}

func (t *sendEDILoadTenderTool) Policy() serviceports.ToolPolicy {
	return ediCreatePolicy(t.Name(),
		"Offers a load to another organization, which may accept it and haul it on the "+
			"terms tendered; a tender is withdrawn, never recalled.")
}

func (t *sendEDILoadTenderTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, fieldShipmentID, permission.ResourceShipment)
}

func (t *sendEDILoadTenderTool) request(
	params *serviceports.ToolExecuteParams,
) (*ediservice.SubmitLoadTenderRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, fieldShipmentID)
	if err != nil {
		return nil, err
	}
	partnerID, _, err := optionalPulid(params.Params, paramEDIPartnerID)
	if err != nil {
		return nil, fmt.Errorf("parameter %q is not a valid id: %w", paramEDIPartnerID, err)
	}

	return &ediservice.SubmitLoadTenderRequest{
		TenantInfo:       tenantFrom(*params),
		SourceShipmentID: shipmentID,
		EDIPartnerID:     partnerID,
	}, nil
}

func (t *sendEDILoadTenderTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.tenders.PlanLoadTender(ctx, req)

	return err
}

func (t *sendEDILoadTenderTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
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

type sendEDIStatusUpdateTool struct {
	documents ediDocumentGenerator
}

var (
	_ serviceports.ToolPreviewer = (*sendEDIStatusUpdateTool)(nil)
	_ serviceports.ToolValidator = (*sendEDIStatusUpdateTool)(nil)
)

func newSendEDIStatusUpdateTool(documents ediDocumentGenerator) serviceports.AgentTool {
	return &sendEDIStatusUpdateTool{documents: documents}
}

func (t *sendEDIStatusUpdateTool) Name() string { return "send_edi_status_update" }

func (t *sendEDIStatusUpdateTool) Description() string {
	return "Propose sending a trading partner an EDI 214 shipment status built from a " +
		"shipment or a service failure. It reports the shipment's current status, or a late " +
		"or missed stop with its reason. Use it when a partner's automatic 214 did not go " +
		"or they ask for one again. The partner's document profile decides the format. An " +
		"invoice's 210 goes from the invoice, not here. A person always decides."
}

func (t *sendEDIStatusUpdateTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramEDIPartnerID: jsonschemautils.Text("The trading partner to send it to, " +
			ediPartnerIDGuidance),
		fieldShipmentID: jsonschemautils.Text("Report this shipment's current status. The " +
			"shipment, from search_shipments or get_shipment."),
		paramServiceFailureID: jsonschemautils.Text("Report a late or missed stop with its " +
			"reason, from list_service_failures or get_service_failure."),
	}, paramEDIPartnerID)
}

func (t *sendEDIStatusUpdateTool) Policy() serviceports.ToolPolicy {
	return ediCreatePolicy(t.Name(),
		"Sends a trading partner a shipment status they act on and pass to their own "+
			"customers; a document sent cannot be recalled.")
}

func (t *sendEDIStatusUpdateTool) request(
	params *serviceports.ToolExecuteParams,
) (*ediservice.GenerateEDIDocumentRequest, error) {
	if err := guardPreview(t, params); err != nil {
		return nil, err
	}

	partnerID, err := requirePulid(params.Params, paramEDIPartnerID)
	if err != nil {
		return nil, err
	}
	req := &ediservice.GenerateEDIDocumentRequest{
		TenantInfo:     tenantFrom(*params),
		EDIPartnerID:   partnerID,
		TransactionSet: edi.TransactionSet214,
		Direction:      edi.DocumentDirectionOutbound,
		GeneratedByID:  params.Actor.UserID,
	}

	sources := 0
	for key, target := range map[string]*pulid.ID{
		fieldShipmentID:       &req.ShipmentID,
		paramServiceFailureID: &req.ServiceFailureID,
	} {
		id, given, idErr := optionalPulid(params.Params, key)
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

func (t *sendEDIStatusUpdateTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.documents.PlanGenerateDocument(ctx, req)

	return err
}

func (t *sendEDIStatusUpdateTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := guardExecute(t, params); err != nil {
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
