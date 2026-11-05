package documentquerytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
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

// recordLabeler names the record a document is attached to the way search
// names it: a shipment by its PRO, a worker by their name.
type recordLabeler interface {
	Labels(
		ctx context.Context,
		req *captureports.ListCaptureRecordLabelsRequest,
	) ([]*captureports.CaptureRecordLabel, error)
}

// uploaderReader finds the person who uploaded a document, in its tenant.
type uploaderReader interface {
	GetByIDs(ctx context.Context, req tenantports.GetUsersByIDsRequest) ([]*tenant.User, error)
}

// getDocumentTool reads a file the person attached, or any document
// they may read: its text, what it looks like, the fields document
// intelligence pulled from it, and where it came from: who uploaded it, when,
// how it came in and the record it is attached to.
type getDocumentTool struct {
	documents documentports.DocumentRepository
	contents  contentReader
	labels    recordLabeler
	users     uploaderReader
	access    fieldgate.Access
}

func NewGetDocumentTool(
	documents documentports.DocumentRepository,
	contents contentReader,
	labels recordLabeler,
	users uploaderReader,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &getDocumentTool{
		documents: documents,
		contents:  contents,
		labels:    labels,
		users:     users,
		access:    fieldgate.NewAccess(permissions),
	}
}

func (t *getDocumentTool) Name() string { return "document_get" }

func (t *getDocumentTool) SearchTerms() []string {
	return []string{
		"read document", "document summary", "read a rate confirmation",
		"who uploaded a document", "where a document came from",
	}
}

func (t *getDocumentTool) Description() string {
	return "Read a document by id: its kind, pages, extracted text and structured fields, " +
		"and where it came from. That is who uploaded it and when, how it came in " +
		"(uploaded by hand, a monitored mailbox, a scan) and the record it is attached " +
		"to, named as people know it. Use it for a document a record names, or for a page document_list " +
		"pointed at; a file the person attached to this conversation is read with " +
		"agent_read_attachment. For a rate confirmation or tender, follow with " +
		"intake_get_draft for the shipment it describes. A document whose reading " +
		"is still pending has no text yet; say so and ask again shortly."
}

func (t *getDocumentTool) ParamSchema() map[string]any {
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

func (t *getDocumentTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), &querytool.ReadSpec{
		Resource: permission.ResourceDocument,
		Reads:    agenttooltrust.ExternalReadAlways,
		Source:   agent.TaintSourceDocument,
		Rationale: "Reads text extracted from a document someone outside sent; nothing changes " +
			"and nothing is sent.",
	})
}

func (t *getDocumentTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	documentID, err := querytool.RequirePulid(params.Params, paramDocumentID)
	if err != nil {
		return nil, err
	}
	window, err := textWindowOf(params.Params)
	if err != nil {
		return nil, err
	}

	tenant := querytool.TenantOf(params)
	doc, err := t.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	readable, err := t.access.ReadableDocuments(ctx, params, []*document.Document{doc},
		func(resource permission.Resource, recordID string) bool {
			return t.access.MayReadRecord(ctx, params, resource, recordID)
		})
	if err != nil {
		return nil, err
	}
	if !readable[doc.ID] {
		return nil, fmt.Errorf(
			"document %s is attached to a %s you may not read",
			doc.ID, textcase.HumanizeSnakeCase(doc.OwnerResource().String()),
		)
	}

	content, err := t.contents.GetContent(ctx, documentID, tenant)
	if err != nil && !IsNotFound(err) {
		return nil, fmt.Errorf("read what was extracted from document %s: %w", doc.ID, err)
	}

	summary, err := summarizeDocument(doc, content, window)
	if err != nil {
		return nil, err
	}
	if err = t.describeOrigin(ctx, tenant, doc, summary); err != nil {
		return nil, err
	}

	return summary, nil
}

// describeOrigin names who uploaded the document and the record it is
// attached to. The caller may read that record (the document is readable
// only then), so its label is theirs to see. A person no longer in the tenant
// is left unnamed rather than guessed.
func (t *getDocumentTool) describeOrigin(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
	summary *documentSummary,
) error {
	if !doc.UploadedByID.IsNil() {
		users, err := t.users.GetByIDs(ctx, tenantports.GetUsersByIDsRequest{
			TenantInfo: tenantInfo,
			UserIDs:    []pulid.ID{doc.UploadedByID},
		})
		if err != nil {
			return fmt.Errorf("read who uploaded document %s: %w", doc.ID, err)
		}
		if len(users) > 0 {
			summary.UploadedBy = strings.TrimSpace(users[0].Name)
		}
	}

	if !capture.IsFileableResource(doc.ResourceType) {
		return nil
	}
	ownerID, err := pulid.Parse(doc.ResourceID)
	if err != nil {
		return nil //nolint:nilerr // a document not on a record by id has no record to name
	}
	labels, err := t.labels.Labels(ctx, &captureports.ListCaptureRecordLabelsRequest{
		TenantInfo:   tenantInfo,
		ResourceType: doc.ResourceType,
		IDs:          []pulid.ID{ownerID},
	})
	if err != nil {
		return fmt.Errorf("name the %s document %s is attached to: %w",
			strings.ReplaceAll(doc.ResourceType, "_", " "), doc.ID, err)
	}
	if len(labels) > 0 {
		summary.AttachedLabel = strings.TrimSpace(labels[0].Title)
	}

	return nil
}
