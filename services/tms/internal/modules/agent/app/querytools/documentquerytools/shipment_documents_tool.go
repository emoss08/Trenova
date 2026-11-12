package documentquerytools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	shipmentResourceType = "shipment"
	maxShipmentDocuments = 50
)

type documentTypeReader interface {
	GetByIDs(
		ctx context.Context,
		req documenttypeports.GetDocumentTypesByIDsRequest,
	) ([]*documenttype.DocumentType, error)
}

// shipmentDocumentsTool lists the papers on file for one shipment by kind.
// An agent asked about a missing proof of delivery could search documents'
// text and read one by id, but could not ask the plain question "what is on
// this shipment", so it could neither confirm the paper was missing nor see
// that it had arrived.
type shipmentDocumentsTool struct {
	documents documentports.DocumentRepository
	types     documentTypeReader
	access    fieldgate.Access
}

func NewShipmentDocumentsTool(
	documents documentports.DocumentRepository,
	types documenttypeports.DocumentTypeRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &shipmentDocumentsTool{
		documents: documents,
		types:     types,
		access:    fieldgate.NewAccess(permissions),
	}
}

func (t *shipmentDocumentsTool) Name() string { return "shipment_list_documents" }

func (t *shipmentDocumentsTool) SearchTerms() []string {
	return []string{
		"documents on a shipment", "papers on file", "proof of delivery on file", "pod missing",
		"is the pod attached", "bill of lading on file", "rate confirmation on file",
		"what documents does this shipment have", "attached documents",
	}
}

func (t *shipmentDocumentsTool) Description() string {
	return "List the documents on file for one shipment: each one's kind (proof of " +
		"delivery, bill of lading, rate confirmation and so on), its status and when it " +
		"was filed. Use it to tell whether a paper is on the shipment or missing, before " +
		"asking anyone for it. A document with no kind was filed without one. It lists " +
		"what is filed, not what billing requires: billing_get_queue_item says what a " +
		"shipment still lacks for billing. Read a document's text with document_get."
}

func (t *shipmentDocumentsTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			querytool.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment's id, from shipment_get, shipment_list or this run's subject."),
		},
		toolschema.KeyRequired:             []string{querytool.ParamShipmentID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *shipmentDocumentsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceShipment,
		Rationale: "Reads which kinds of document are filed on a shipment and when; it " +
			"returns no document's text or file name, changes nothing and sends nothing.",
	})
}

type shipmentDocument struct {
	DocumentID string `json:"documentId"`
	Kind       string `json:"kind"`
	KindCode   string `json:"kindCode,omitempty"`
	Status     string `json:"status"`
	FiledAt    int64  `json:"filedAt"`
}

type shipmentDocuments struct {
	ShipmentID string             `json:"shipmentId"`
	Documents  []shipmentDocument `json:"documents"`
	// More is set when the shipment has more documents than one read lists.
	More bool   `json:"more,omitempty"`
	Note string `json:"note,omitempty"`
}

func (t *shipmentDocumentsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	shipmentID, err := querytool.RequirePulid(params.Params, querytool.ParamShipmentID)
	if err != nil {
		return nil, err
	}
	if !t.access.MayReadRecord(ctx, params, permission.ResourceShipment, shipmentID.String()) {
		return nil, fmt.Errorf("shipment %s is not one you may read", shipmentID)
	}

	tenant := querytool.TenantOf(params)
	listed, err := t.documents.List(ctx, &documentports.ListDocumentsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: maxShipmentDocuments},
		},
		ResourceID:   shipmentID.String(),
		ResourceType: shipmentResourceType,
	})
	if err != nil {
		return nil, fmt.Errorf("list the documents on shipment %s: %w", shipmentID, err)
	}

	kinds, err := t.kindsOf(ctx, tenant, listed.Items)
	if err != nil {
		return nil, err
	}

	out := &shipmentDocuments{
		ShipmentID: shipmentID.String(),
		Documents:  make([]shipmentDocument, 0, len(listed.Items)),
		More:       listed.Total > len(listed.Items),
	}
	for _, doc := range listed.Items {
		if doc == nil || doc.Status == document.StatusRejected ||
			doc.Status == document.StatusArchived {
			continue
		}
		row := shipmentDocument{
			DocumentID: doc.ID.String(),
			Kind:       "No kind set",
			Status:     string(doc.Status),
			FiledAt:    doc.CreatedAt,
		}
		if doc.DocumentTypeID != nil {
			if kind, ok := kinds[*doc.DocumentTypeID]; ok {
				row.Kind, row.KindCode = kind.Name, kind.Code
			}
		}
		out.Documents = append(out.Documents, row)
	}
	if len(out.Documents) == 0 {
		out.Note = "No document is on file for this shipment."
	}

	return out, nil
}

func (t *shipmentDocumentsTool) kindsOf(
	ctx context.Context,
	tenant pagination.TenantInfo,
	docs []*document.Document,
) (map[pulid.ID]*documenttype.DocumentType, error) {
	ids := make([]pulid.ID, 0, len(docs))
	seen := make(map[pulid.ID]struct{}, len(docs))
	for _, doc := range docs {
		if doc == nil || doc.DocumentTypeID == nil || doc.DocumentTypeID.IsNil() {
			continue
		}
		if _, dup := seen[*doc.DocumentTypeID]; !dup {
			seen[*doc.DocumentTypeID] = struct{}{}
			ids = append(ids, *doc.DocumentTypeID)
		}
	}
	out := make(map[pulid.ID]*documenttype.DocumentType, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	found, err := t.types.GetByIDs(ctx, documenttypeports.GetDocumentTypesByIDsRequest{
		TenantInfo:      tenant,
		DocumentTypeIDs: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("read the kinds of a shipment's documents: %w", err)
	}
	for _, kind := range found {
		if kind != nil {
			out[kind.ID] = kind
		}
	}

	return out, nil
}
