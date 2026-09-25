package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/documentservice"
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

type attachDocumentTool struct {
	documents documentAttacher
}

func newAttachDocumentTool(documents documentAttacher) serviceports.AgentTool {
	return &attachDocumentTool{documents: documents}
}

func (t *attachDocumentTool) Name() string { return "attach_document_to_shipment" }

func (t *attachDocumentTool) Description() string {
	return "Attach a document that already exists to a shipment, so it rides with the load " +
		"and its billing. Use it for a POD, a rate confirmation or a signed bill that arrived " +
		"on its own and belongs to a shipment you have identified. Find the document first — " +
		"get_document_summary and the document searches return its id. This moves an existing " +
		"file; it cannot upload one, so a document that is not in the system yet has to be " +
		"uploaded by a person before it can be attached."
}

func (t *attachDocumentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"documentId": map[string]any{
				"type":        "string",
				"description": "The document to attach, from get_document_summary or a search.",
			},
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment it belongs to, from the page, list_shipments or " +
					"search_shipments.",
			},
		},
		"required":             []string{"documentId", "shipmentId"},
		"additionalProperties": false,
	}
}

func (t *attachDocumentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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
func (t *attachDocumentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "documentId", permission.ResourceDocument)
}

func (t *attachDocumentTool) Validate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) error {
	return t.validateArgs(params.Params)
}

func (t *attachDocumentTool) validateArgs(params map[string]any) error {
	if _, err := requirePulid(params, "documentId"); err != nil {
		return err
	}
	if _, err := requirePulid(params, "shipmentId"); err != nil {
		return err
	}

	return nil
}

func (t *attachDocumentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := t.request(params)
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

func (t *attachDocumentTool) request(
	params serviceports.ToolExecuteParams,
) (*documentservice.AttachLineageRequest, error) {
	documentID, err := requirePulid(params.Params, "documentId")
	if err != nil {
		return nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
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
