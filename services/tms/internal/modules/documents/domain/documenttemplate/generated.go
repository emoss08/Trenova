package documenttemplate

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// defaultGeneratedMimeType matches the column default, so a record validated
// before insert is judged against the value the insert would have stored.
const defaultGeneratedMimeType = "application/pdf"

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
	keel.Table `keel:"generated_documents"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:gd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Kind Kind `json:"kind" keel:"kind"`

	// TemplateID and TemplateVersionID are nil when the built-in rendered, since a
	// built-in has no database row. ContentHash still identifies the exact content.
	TemplateID        *pulid.ID `json:"templateId" keel:"template_id"`
	TemplateVersionID *pulid.ID `json:"templateVersionId" keel:"template_version_id"`

	// DocumentID links to the stored document row the bytes were uploaded as, so a
	// generated artifact is reachable from the normal document surfaces.
	DocumentID *pulid.ID `json:"documentId" keel:"document_id"`

	ReferenceType string   `json:"referenceType" keel:"reference_type"`
	ReferenceID   pulid.ID `json:"referenceId" keel:"reference_id"`

	FileName string `json:"fileName" keel:"file_name"`
	FilePath string `json:"filePath" keel:"file_path,nullzero"`
	FileSize int64  `json:"fileSize" keel:"file_size"`
	MimeType string `json:"mimeType" keel:"mime_type"`
	Checksum string `json:"checksum" keel:"checksum,nullzero"`

	// ContentHash is the template content that produced this, which survives even
	// when the version row does not.
	ContentHash string `json:"contentHash" keel:"content_hash,nullzero"`

	// Warnings records what the render dropped — a blocked image, an exceeded asset
	// budget — so an omission is never silent.
	Warnings []GeneratedDocumentWarning `json:"warnings" keel:"warnings,jsonb,nullzero"`

	Status        GenerationStatus `json:"status" keel:"status"`
	ErrorMessage  string           `json:"errorMessage" keel:"error_message,nullzero"`
	GeneratedAt   *int64           `json:"generatedAt" keel:"generated_at"`
	GeneratedByID *pulid.ID        `json:"generatedById" keel:"generated_by_id"`

	DeliveryMethod DeliveryMethod `json:"deliveryMethod" keel:"delivery_method"`
	DeliveredAt    *int64         `json:"deliveredAt" keel:"delivered_at"`
	DeliveredTo    string         `json:"deliveredTo" keel:"delivered_to,nullzero"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	BusinessUnit    *tenant.BusinessUnit     `json:"-"`
	Organization    *tenant.Organization     `json:"-"`
	Template        *DocumentTemplate        `json:"template,omitempty" keel:"belongs:template_id"`
	TemplateVersion *DocumentTemplateVersion `json:"templateVersion,omitempty" keel:"belongs:template_version_id"`
}

func (g *GeneratedDocument) BeforeUpdate(_ context.Context) error {
	g.Normalize()

	return nil
}

func (g *GeneratedDocument) BeforeCreate(_ context.Context) error {
	g.Normalize()
	if g.MimeType == "" {
		g.MimeType = "application/pdf"
	}
	if g.Status == "" {
		g.Status = GenerationStatusPending
	}
	if g.DeliveryMethod == "" {
		g.DeliveryMethod = DeliveryMethodNone
	}

	return nil
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
	g.Normalize()

	multiErr.AddOzzoError(validation.ValidateStruct(g,
		validation.Field(&g.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&g.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&g.Kind,
			validation.Required.Error("Template kind is required"),
			validation.Length(1, MaxKindLength).
				Error("Template kind cannot be longer than 100 characters"),
		),
		validation.Field(&g.ReferenceType,
			validation.Required.Error("Reference type is required"),
			validation.Length(1, MaxReferenceTypeLength).
				Error("Reference type cannot be longer than 50 characters"),
		),
		validation.Field(&g.ReferenceID, validation.Required.Error("Reference is required")),
		validation.Field(&g.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, MaxFileNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&g.FilePath,
			validation.Length(0, MaxFilePathLength).
				Error("File path cannot be longer than 500 characters"),
			// These mirror database CHECK constraints. Catching them here turns an
			// opaque constraint violation into a field error a caller can act on.
			validation.When(g.Status == GenerationStatusCompleted, validation.Required.Error(
				"A completed document must record where it was stored",
			)),
		),
		validation.Field(&g.FileSize,
			validation.Min(int64(0)).Error("File size cannot be negative"),
		),
		validation.Field(&g.MimeType,
			validation.Required.Error("Mime type is required"),
			validation.Length(1, MaxMimeTypeLength).
				Error("Mime type cannot be longer than 100 characters"),
		),
		validation.Field(&g.Checksum,
			validation.Length(0, MaxHashLength).
				Error("Checksum cannot be longer than 64 characters"),
		),
		validation.Field(&g.ContentHash,
			validation.Length(0, MaxHashLength).
				Error("Content hash cannot be longer than 64 characters"),
		),
		validation.Field(&g.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[GenerationStatus](
				"Status is not a valid generation status",
			),
		),
		validation.Field(&g.GeneratedAt,
			validation.When(g.Status == GenerationStatusCompleted, validation.Required.Error(
				"A completed document must record when it was generated",
			)),
		),
		validation.Field(&g.ErrorMessage,
			validation.When(g.Status == GenerationStatusFailed, validation.Required.Error(
				"A failed document must record why",
			)),
		),
		validation.Field(&g.DeliveryMethod,
			validation.Required.Error("Delivery method is required"),
			domainvalidation.ValidEnum[DeliveryMethod]("Delivery method is not supported"),
		),
		validation.Field(&g.DeliveredTo,
			validation.Length(0, MaxDeliveredToLength).
				Error("Recipient cannot be longer than 255 characters"),
			validation.When(g.DeliveredAt != nil, validation.Required.Error(
				"A delivered document must record who received it",
			)),
		),
		validation.Field(&g.DeliveredAt,
			validation.When(g.DeliveredTo != "", validation.Required.Error(
				"A delivered document must record when it was delivered",
			)),
		),
	))
}

// Normalize trims the free text the render path fills in and applies the column
// defaults.
//
// The defaults are applied here rather than only at insert because validation
// runs first: a caller that leaves them unset would otherwise fail a check the
// insert itself would have satisfied.
func (g *GeneratedDocument) Normalize() {
	g.ReferenceType = strings.TrimSpace(g.ReferenceType)
	g.FileName = strings.TrimSpace(g.FileName)
	g.FilePath = strings.TrimSpace(g.FilePath)
	g.ErrorMessage = strings.TrimSpace(g.ErrorMessage)
	g.DeliveredTo = strings.TrimSpace(g.DeliveredTo)

	if g.MimeType == "" {
		g.MimeType = defaultGeneratedMimeType
	}
	if g.DeliveryMethod == "" {
		g.DeliveryMethod = DeliveryMethodNone
	}
	if g.Status == "" {
		g.Status = GenerationStatusPending
	}
}

func (g *GeneratedDocument) BeforeSave(_ context.Context) error {
	g.Normalize()
	return nil
}
