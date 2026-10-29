package agentquerytoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// contentReader is the one question the summary tool asks of document
// intelligence: what has it made of this file.
type contentReader interface {
	GetContent(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
}

// getDocumentSummaryTool reads a file the person attached, or any document
// they may read: its text, what it looks like, and the fields document
// intelligence pulled from it.
type getDocumentSummaryTool struct {
	documents documentports.DocumentRepository
	contents  contentReader
	access    fieldAccess
}

func newGetDocumentSummaryTool(
	documents documentports.DocumentRepository,
	contents contentReader,
	permissions permissionports.PermissionEngine,
	threads assistantports.ThreadOwnerRepository,
) agenttoolports.AgentQueryTool {
	return &getDocumentSummaryTool{
		documents: documents,
		contents:  contents,
		access:    newFieldAccess(permissions).withThreads(threads),
	}
}

func (t *getDocumentSummaryTool) Name() string { return "document_get_summary" }

func (t *getDocumentSummaryTool) SearchTerms() []string {
	return []string{"read document", "document summary", "read a rate confirmation"}
}

func (t *getDocumentSummaryTool) Description() string {
	return "Read a document by id: what kind of document it looks like, how many pages, " +
		"the text extracted from it, and the structured fields document intelligence " +
		"found. Use it for a document a record names, or for a page document_list " +
		"pointed at; a file the person attached to this conversation is read with " +
		"agent_read_attachment. For a rate confirmation or tender, follow with " +
		"intake_get_draft for the shipment it describes. A document whose reading " +
		"is still pending has no text yet; say so and ask again shortly."
}

func (t *getDocumentSummaryTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: withTextWindow(map[string]any{
			paramDocumentID: agenttoolschema.RecordIDText(permission.ResourceDocument,
				"The document's id, from a documentId in inbox_get_inbound_message or "+
					"worker_get_credential, a document_list result, or this run's subject."),
		}),
		toolschema.KeyRequired:             []string{paramDocumentID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *getDocumentSummaryTool) Policy() agenttoolports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceDocument,
		reads:    agenttooltrust.ExternalReadAlways,
		source:   agent.TaintSourceDocument,
		rationale: "Reads text extracted from a document someone outside sent; nothing changes " +
			"and nothing is sent.",
	})
}

func (t *getDocumentSummaryTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	documentID, err := requirePulid(params.Params, paramDocumentID)
	if err != nil {
		return nil, err
	}
	window, err := textWindowOf(params.Params)
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	doc, err := t.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	readable, err := t.access.readableDocuments(ctx, params, []*document.Document{doc},
		func(resource permission.Resource, recordID string) bool {
			return t.access.mayReadRecord(ctx, params, resource, recordID)
		})
	if err != nil {
		return nil, err
	}
	if !readable[doc.ID] {
		return nil, fmt.Errorf(
			"document %s is attached to a %s you may not read",
			doc.ID, stringutils.HumanizeSnakeCase(doc.OwnerResource().String()),
		)
	}

	content, err := t.contents.GetContent(ctx, documentID, tenant)
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("read what was extracted from document %s: %w", doc.ID, err)
	}

	return summarizeDocument(doc, content, window)
}
