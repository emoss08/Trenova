package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListDocumentTemplatesRequest struct {
	Filter *pagination.QueryOptions
	// Kind narrows to one template kind, which is how the admin catalog asks "what
	// has this organization customized for invoices".
	Kind documenttemplate.Kind
}

type ListDocumentTemplateConnectionRequest struct {
	Filter                  *pagination.QueryOptions `json:"filter"`
	Cursor                  pagination.CursorInfo    `json:"-"`
	DocumentTemplateColumns []string                 `json:"-"`
}

type GetDocumentTemplateByIDRequest struct {
	DocumentTemplateID pulid.ID
	TenantInfo         pagination.TenantInfo
	// IncludeVersions loads the full history, newest first, for the version rail.
	IncludeVersions bool
	// IncludeActiveVersion loads only what currently renders, which is all a render
	// path needs.
	IncludeActiveVersion bool
	IncludeAssignments   bool
}

type GetDocumentTemplateVersionByIDRequest struct {
	VersionID  pulid.ID
	TenantInfo pagination.TenantInfo
}

type ListDocumentTemplateVersionsRequest struct {
	TemplateID pulid.ID
	TenantInfo pagination.TenantInfo
}

// ResolveDocumentTemplateRequest asks which template governs one render.
//
// CustomerID is optional: a kind with no customer in scope, such as a scheduled
// report, passes it nil and the assignment tier is skipped entirely.
type ResolveDocumentTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind
	CustomerID *pulid.ID
}

// ResolvedTemplateRow is what the resolution query returns: the active version plus
// which tier of the chain produced it.
type ResolvedTemplateRow struct {
	Template *documenttemplate.DocumentTemplate
	Version  *documenttemplate.DocumentTemplateVersion
	Source   documenttemplate.TemplateSource
}

type CreateDocumentTemplateVersionRequest struct {
	Version *documenttemplate.DocumentTemplateVersion
	// SourceVersionID copies content from an existing version, which is how both
	// "edit a copy of what is live" and rollback are expressed.
	SourceVersionID *pulid.ID
}

// PublishDocumentTemplateVersionRequest makes one version live.
//
// The repository performs the swap in a single transaction: archive whatever was
// active, mark this one active, and point the template at it. Doing it in steps
// would leave a window in which a render finds no active version and silently falls
// back to the built-in.
type PublishDocumentTemplateVersionRequest struct {
	TenantInfo    pagination.TenantInfo
	VersionID     pulid.ID
	PublishedByID pulid.ID
	PublishNotes  string
}

type ArchiveDocumentTemplateVersionRequest struct {
	TenantInfo   pagination.TenantInfo
	VersionID    pulid.ID
	ArchivedByID pulid.ID
}

type ListDocumentTemplateAssignmentsRequest struct {
	Filter     *pagination.QueryOptions
	CustomerID *pulid.ID
	Kind       documenttemplate.Kind
}

type ListDocumentTemplateAssignmentConnectionRequest struct {
	Filter                            *pagination.QueryOptions `json:"filter"`
	Cursor                            pagination.CursorInfo    `json:"-"`
	DocumentTemplateAssignmentColumns []string                 `json:"-"`
}

type GetCustomerAssignmentsRequest struct {
	TenantInfo pagination.TenantInfo
	CustomerID pulid.ID
}

type UnassignDocumentTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       documenttemplate.Kind
	CustomerID pulid.ID
}

type DocumentTemplateSelectOptionsRequest struct {
	SelectQueryRequest *pagination.SelectQueryRequest
	Kind               documenttemplate.Kind
}

type DocumentTemplateRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentTemplatesRequest,
	) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error)

	ListConnection(
		ctx context.Context,
		req *ListDocumentTemplateConnectionRequest,
	) (*pagination.CursorListResult[*documenttemplate.DocumentTemplate], error)

	GetByID(
		ctx context.Context,
		req *GetDocumentTemplateByIDRequest,
	) (*documenttemplate.DocumentTemplate, error)

	// Resolve returns the template governing a render, or nil when the organization
	// has customized nothing for this kind and the built-in should be used.
	Resolve(
		ctx context.Context,
		req *ResolveDocumentTemplateRequest,
	) (*ResolvedTemplateRow, error)

	Create(
		ctx context.Context,
		entity *documenttemplate.DocumentTemplate,
	) (*documenttemplate.DocumentTemplate, error)

	Update(
		ctx context.Context,
		entity *documenttemplate.DocumentTemplate,
	) (*documenttemplate.DocumentTemplate, error)

	Delete(ctx context.Context, req *GetDocumentTemplateByIDRequest) error

	SelectOptions(
		ctx context.Context,
		req *DocumentTemplateSelectOptionsRequest,
	) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error)
}

type DocumentTemplateVersionRepository interface {
	GetByID(
		ctx context.Context,
		req *GetDocumentTemplateVersionByIDRequest,
	) (*documenttemplate.DocumentTemplateVersion, error)

	List(
		ctx context.Context,
		req *ListDocumentTemplateVersionsRequest,
	) ([]*documenttemplate.DocumentTemplateVersion, error)

	Create(
		ctx context.Context,
		req *CreateDocumentTemplateVersionRequest,
	) (*documenttemplate.DocumentTemplateVersion, error)

	// Update only ever succeeds on a draft; the service refuses a published version
	// before it gets here, and the repository refuses again so a new caller cannot
	// bypass the rule.
	Update(
		ctx context.Context,
		entity *documenttemplate.DocumentTemplateVersion,
	) (*documenttemplate.DocumentTemplateVersion, error)

	Publish(
		ctx context.Context,
		req *PublishDocumentTemplateVersionRequest,
	) (*documenttemplate.DocumentTemplateVersion, error)

	Archive(
		ctx context.Context,
		req *ArchiveDocumentTemplateVersionRequest,
	) (*documenttemplate.DocumentTemplateVersion, error)

	Delete(ctx context.Context, req *GetDocumentTemplateVersionByIDRequest) error
}

type DocumentTemplateAssignmentRepository interface {
	List(
		ctx context.Context,
		req *ListDocumentTemplateAssignmentsRequest,
	) (*pagination.ListResult[*documenttemplate.DocumentTemplateAssignment], error)

	ListConnection(
		ctx context.Context,
		req *ListDocumentTemplateAssignmentConnectionRequest,
	) (*pagination.CursorListResult[*documenttemplate.DocumentTemplateAssignment], error)

	GetForCustomer(
		ctx context.Context,
		req *GetCustomerAssignmentsRequest,
	) ([]*documenttemplate.DocumentTemplateAssignment, error)

	// Assign is an upsert: one assignment per customer per kind, so re-assigning
	// replaces rather than accumulating rows the resolution query would have to
	// break a tie between.
	Assign(
		ctx context.Context,
		entity *documenttemplate.DocumentTemplateAssignment,
	) (*documenttemplate.DocumentTemplateAssignment, error)

	Unassign(ctx context.Context, req *UnassignDocumentTemplateRequest) error
}

type ListGeneratedDocumentsRequest struct {
	Filter        *pagination.QueryOptions
	Kind          documenttemplate.Kind
	ReferenceType string
	ReferenceID   *pulid.ID
}

type GetGeneratedDocumentByIDRequest struct {
	GeneratedDocumentID pulid.ID
	TenantInfo          pagination.TenantInfo
}

type GeneratedDocumentRepository interface {
	List(
		ctx context.Context,
		req *ListGeneratedDocumentsRequest,
	) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error)

	GetByID(
		ctx context.Context,
		req *GetGeneratedDocumentByIDRequest,
	) (*documenttemplate.GeneratedDocument, error)

	Create(
		ctx context.Context,
		entity *documenttemplate.GeneratedDocument,
	) (*documenttemplate.GeneratedDocument, error)

	Update(
		ctx context.Context,
		entity *documenttemplate.GeneratedDocument,
	) (*documenttemplate.GeneratedDocument, error)
}
