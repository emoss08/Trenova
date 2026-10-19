package masterdatatools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const documentResourceTypeShipment = "shipment"

// documentAttacher is the sliver of the document service this needs. Attaching
// is a lineage move on a document that already exists, which is a different
// operation from uploading one and belongs to a different service.
type documentAttacher interface {
	PreviewAttachLineage(
		ctx context.Context,
		req *documentservice.AttachLineageRequest,
	) (*documentservice.AttachLineagePlan, error)
	AttachLineageToResource(
		ctx context.Context,
		documentID pulid.ID,
		resourceType string,
		resourceID string,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*document.Document, error)
}

type AttachDocumentTool struct {
	documents documentAttacher
}

func NewAttachDocumentTool(documents documentAttacher) agentports.AgentTool {
	return &AttachDocumentTool{documents: documents}
}

func (t *AttachDocumentTool) Name() string { return "shipment_attach_document" }

func (t *AttachDocumentTool) Recipe() []string {
	return []string{"get_inbound_message", "shipment_list", "shipment_attach_document"}
}

func (t *AttachDocumentTool) Description() string {
	return "Attach a document that already exists to a shipment, so it rides with the load " +
		"and its billing. Use it for a POD, a rate confirmation or a signed bill that arrived " +
		"on its own and belongs to a shipment you have identified. Find the document first — " +
		"get_document_summary and the document searches return its id. This moves an existing " +
		"file; it cannot upload one, so a document that is not in the system yet has to be " +
		"uploaded by a person before it can be attached."
}

func (t *AttachDocumentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"documentId": agenttoolschema.RecordIDText(
				permission.ResourceDocument,
				"The document to attach, from get_document_summary or a search.",
			),
			"shipmentId": agenttoolschema.RecordIDText(
				permission.ResourceShipment,
				"The shipment it belongs to, from the page, shipment_list or "+
					"shipment_list.",
			),
		},
		"required":             []string{"documentId", "shipmentId"},
		"additionalProperties": false,
	}
}

func (t *AttachDocumentTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDocument,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Files a document already in Trenova against a shipment; nobody outside " +
			"is told.",
	}
}

// Target names the record this changes, so the proposal is pinned to the
// document's version and a file somebody moved in the meantime cannot be moved
// again on stale information.
func (t *AttachDocumentTool) Target(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, "documentId", permission.ResourceDocument)
}

func (t *AttachDocumentTool) Validate(
	_ context.Context,
	params agentports.ToolExecuteParams,
) error {
	return t.validateArgs(params.Params)
}

func (t *AttachDocumentTool) validateArgs(params map[string]any) error {
	if _, err := common.RequirePulid(params, "documentId"); err != nil {
		return err
	}
	if _, err := common.RequirePulid(params, "shipmentId"); err != nil {
		return err
	}

	return nil
}

func (t *AttachDocumentTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.documents.AttachLineageToResource(
		ctx,
		request.DocumentID,
		request.ResourceType,
		request.ResourceID,
		request.TenantInfo,
		params.Actor.UserID,
	)

	return err
}

func (t *AttachDocumentTool) request(
	params *agentports.ToolExecuteParams,
) (*documentservice.AttachLineageRequest, error) {
	documentID, err := common.RequirePulid(params.Params, "documentId")
	if err != nil {
		return nil, err
	}

	shipmentID, err := common.RequirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}

	return &documentservice.AttachLineageRequest{
		DocumentID:   documentID,
		ResourceType: documentResourceTypeShipment,
		ResourceID:   shipmentID.String(),
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
	}, nil
}
