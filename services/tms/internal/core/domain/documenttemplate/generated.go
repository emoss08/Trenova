package documenttemplate

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

// GenerationStatus tracks one rendering attempt.
type GenerationStatus string

const (
	GenerationStatusPending    = GenerationStatus("Pending")
	GenerationStatusProcessing = GenerationStatus("Processing")
	GenerationStatusCompleted  = GenerationStatus("Completed")
	GenerationStatusFailed     = GenerationStatus("Failed")
)

func (s GenerationStatus) String() string { return string(s) }

func (s GenerationStatus) IsValid() bool {
	switch s {
	case GenerationStatusPending, GenerationStatusProcessing,
		GenerationStatusCompleted, GenerationStatusFailed:
		return true
	default:
		return false
	}
}

func GenerationStatusFromString(v string) (GenerationStatus, error) {
	switch v {
	case "Pending":
		return GenerationStatusPending, nil
	case "Processing":
		return GenerationStatusProcessing, nil
	case "Completed":
		return GenerationStatusCompleted, nil
	case "Failed":
		return GenerationStatusFailed, nil
	default:
		return "", errors.New("invalid generated document status")
	}
}

// DeliveryMethod records how a rendered document reached its recipient.
type DeliveryMethod string

const (
	DeliveryMethodNone     = DeliveryMethod("None")
	DeliveryMethodEmail    = DeliveryMethod("Email")
	DeliveryMethodDownload = DeliveryMethod("Download")
	DeliveryMethodPortal   = DeliveryMethod("Portal")
)

func (m DeliveryMethod) String() string { return string(m) }

func (m DeliveryMethod) IsValid() bool {
	switch m {
	case DeliveryMethodNone, DeliveryMethodEmail,
		DeliveryMethodDownload, DeliveryMethodPortal:
		return true
	default:
		return false
	}
}

func DeliveryMethodFromString(v string) (DeliveryMethod, error) {
	switch v {
	case "None":
		return DeliveryMethodNone, nil
	case "Email":
		return DeliveryMethodEmail, nil
	case "Download":
		return DeliveryMethodDownload, nil
	case "Portal":
		return DeliveryMethodPortal, nil
	default:
		return "", errors.New("invalid delivery method")
	}
}

var (
	_ bun.BeforeAppendModelHook          = (*GeneratedDocument)(nil)
	_ validationframework.TenantedEntity = (*GeneratedDocument)(nil)
	_ domaintypes.PostgresSearchable     = (*GeneratedDocument)(nil)
)

// GeneratedDocument is one rendered artifact and the template version that produced
// it.
//
// TemplateVersionID is the point of this table. Without it, "what did we actually
// send that customer in March" is unanswerable once a template has been edited —
// and a detention charge or an invoice dispute is exactly when that question gets
// asked. The foreign key is RESTRICT so a version with output attributed to it
// cannot be deleted out from under the record.
type GeneratedDocument struct {
	bun.BaseModel             `bun:"table:generated_documents,alias:gd" json:"-"`
	pagination.CursorValueSet `bun:",embed"                             json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Kind Kind `json:"kind" bun:"kind,type:VARCHAR(100),notnull"`

	// TemplateID and TemplateVersionID are nil when the built-in rendered, since a
	// built-in has no database row. ContentHash still identifies the exact content.
	TemplateID        *pulid.ID `json:"templateId"        bun:"template_id,type:VARCHAR(100),nullzero"`
	TemplateVersionID *pulid.ID `json:"templateVersionId" bun:"template_version_id,type:VARCHAR(100),nullzero"`

	// DocumentID links to the stored document row the bytes were uploaded as, so a
	// generated artifact is reachable from the normal document surfaces.
	DocumentID *pulid.ID `json:"documentId" bun:"document_id,type:VARCHAR(100),nullzero"`

	ReferenceType string   `json:"referenceType" bun:"reference_type,type:VARCHAR(50),notnull"`
	ReferenceID   pulid.ID `json:"referenceId"   bun:"reference_id,type:VARCHAR(100),notnull"`

	FileName string `json:"fileName" bun:"file_name,type:VARCHAR(255),notnull"`
	FilePath string `json:"filePath" bun:"file_path,type:VARCHAR(500),nullzero"`
	FileSize int64  `json:"fileSize" bun:"file_size,type:BIGINT,notnull,default:0"`
	MimeType string `json:"mimeType" bun:"mime_type,type:VARCHAR(100),notnull,default:'application/pdf'"`
	Checksum string `json:"checksum" bun:"checksum,type:VARCHAR(64),nullzero"`

	// ContentHash is the template content that produced this, which survives even
	// when the version row does not.
	ContentHash string `json:"contentHash" bun:"content_hash,type:VARCHAR(64),nullzero"`

	// Warnings records what the render dropped — a blocked image, an exceeded asset
	// budget — so an omission is never silent.
	Warnings []GeneratedDocumentWarning `json:"warnings" bun:"warnings,type:JSONB,nullzero"`

	Status        GenerationStatus `json:"status"        bun:"status,type:generated_document_status_enum,notnull,default:'Pending'"`
	ErrorMessage  string           `json:"errorMessage"  bun:"error_message,type:TEXT,nullzero"`
	GeneratedAt   *int64           `json:"generatedAt"   bun:"generated_at,type:BIGINT,nullzero"`
	GeneratedByID *pulid.ID        `json:"generatedById" bun:"generated_by_id,type:VARCHAR(100),nullzero"`

	DeliveryMethod DeliveryMethod `json:"deliveryMethod" bun:"delivery_method,type:generated_document_delivery_enum,notnull,default:'None'"`
	DeliveredAt    *int64         `json:"deliveredAt"    bun:"delivered_at,type:BIGINT,nullzero"`
	DeliveredTo    string         `json:"deliveredTo"    bun:"delivered_to,type:VARCHAR(255),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit    *tenant.BusinessUnit     `json:"-"                         bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization     `json:"-"                         bun:"rel:belongs-to,join:organization_id=id"`
	Template        *DocumentTemplate        `json:"template,omitempty"        bun:"rel:belongs-to,join:template_id=id"`
	TemplateVersion *DocumentTemplateVersion `json:"templateVersion,omitempty" bun:"rel:belongs-to,join:template_version_id=id"`
}

// GeneratedDocumentWarning is one non-fatal problem from a render.
type GeneratedDocumentWarning struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (g *GeneratedDocument) GetID() pulid.ID { return g.ID }

func (g *GeneratedDocument) GetCreatedAt() int64 { return g.CreatedAt }

func (g *GeneratedDocument) GetOrganizationID() pulid.ID { return g.OrganizationID }

func (g *GeneratedDocument) GetBusinessUnitID() pulid.ID { return g.BusinessUnitID }

func (g *GeneratedDocument) GetTableName() string { return "generated_documents" }

// GetPostgresSearchConfig mirrors the generated column the migration builds, so a
// search here uses the index rather than scanning.
func (g *GeneratedDocument) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "gd",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "reference_type",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

// MarkCompleted records a successful render.
func (g *GeneratedDocument) MarkCompleted(filePath string, size int64, checksum string) {
	now := timeutils.NowUnix()

	g.Status = GenerationStatusCompleted
	g.FilePath = filePath
	g.FileSize = size
	g.Checksum = checksum
	g.GeneratedAt = &now
	g.ErrorMessage = ""
}

// MarkFailed records why a render did not produce a document.
func (g *GeneratedDocument) MarkFailed(reason string) {
	g.Status = GenerationStatusFailed
	// The database requires a reason on a failed row, because "it failed" with no
	// explanation is a support ticket nobody can close.
	if strings.TrimSpace(reason) == "" {
		reason = "rendering failed without a reported reason"
	}
	g.ErrorMessage = reason
}

// MarkDelivered records that the artifact reached someone.
func (g *GeneratedDocument) MarkDelivered(method DeliveryMethod, recipient string) {
	now := timeutils.NowUnix()

	g.DeliveryMethod = method
	g.DeliveredAt = &now
	g.DeliveredTo = recipient
}

// FromBuiltIn reports whether this was rendered from an embedded starter rather than
// an organization's own template.
func (g *GeneratedDocument) FromBuiltIn() bool {
	return g.TemplateVersionID == nil || g.TemplateVersionID.IsNil()
}

func (g *GeneratedDocument) Validate(multiErr *errortypes.MultiError) {
	if g.Kind == "" {
		multiErr.Add("kind", errortypes.ErrRequired, "Template kind is required")
	}

	if strings.TrimSpace(g.ReferenceType) == "" {
		multiErr.Add("referenceType", errortypes.ErrRequired, "Reference type is required")
	}

	if g.ReferenceID.IsNil() {
		multiErr.Add("referenceId", errortypes.ErrRequired, "Reference is required")
	}

	if strings.TrimSpace(g.FileName) == "" {
		multiErr.Add("fileName", errortypes.ErrRequired, "File name is required")
	}

	if !g.Status.IsValid() {
		multiErr.Add("status", errortypes.ErrInvalid, "Status is not a valid generation status")
	}

	if !g.DeliveryMethod.IsValid() {
		multiErr.Add("deliveryMethod", errortypes.ErrInvalid, "Delivery method is not supported")
	}

	// These mirror database CHECK constraints. Catching them here turns an opaque
	// constraint violation into a field error a caller can act on.
	if g.Status == GenerationStatusCompleted {
		if g.GeneratedAt == nil {
			multiErr.Add("generatedAt", errortypes.ErrRequired,
				"A completed document must record when it was generated")
		}
		if strings.TrimSpace(g.FilePath) == "" {
			multiErr.Add("filePath", errortypes.ErrRequired,
				"A completed document must record where it was stored")
		}
	}

	if g.Status == GenerationStatusFailed && strings.TrimSpace(g.ErrorMessage) == "" {
		multiErr.Add("errorMessage", errortypes.ErrRequired,
			"A failed document must record why")
	}
}

func (g *GeneratedDocument) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if g.MimeType == "" {
		g.MimeType = "application/pdf"
	}
	if g.DeliveryMethod == "" {
		g.DeliveryMethod = DeliveryMethodNone
	}
	if g.Status == "" {
		g.Status = GenerationStatusPending
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if g.ID.IsNil() {
			g.ID = pulid.MustNew("gd_")
		}
		g.CreatedAt = now
		g.UpdatedAt = now
	case *bun.UpdateQuery:
		g.UpdatedAt = now
	}

	return nil
}
