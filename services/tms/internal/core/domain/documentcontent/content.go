package documentcontent

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Status string
type SourceKind string

const (
	StatusPending    Status = "Pending"
	StatusExtracting Status = "Extracting"
	StatusExtracted  Status = "Extracted"
	StatusIndexed    Status = "Indexed"
	StatusFailed     Status = "Failed"
)

const (
	SourceKindNative SourceKind = "native_text"
	SourceKindOCR    SourceKind = "ocr"
	SourceKindMixed  SourceKind = "mixed"
)

type Content struct {
	bun.BaseModel `bun:"table:document_contents,alias:dc" json:"-"`

	ID                       pulid.ID       `json:"id"                       bun:"id,type:VARCHAR(100),pk,notnull"`
	DocumentID               pulid.ID       `json:"documentId"               bun:"document_id,type:VARCHAR(100),notnull"`
	OrganizationID           pulid.ID       `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID           pulid.ID       `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	Status                   Status         `json:"status"                   bun:"status,type:document_content_status_enum,notnull,default:'Pending'"`
	ContentText              string         `json:"contentText"              bun:"content_text,type:TEXT,nullzero"`
	PageCount                int            `json:"pageCount"                bun:"page_count,type:INTEGER,notnull,default:0"`
	SourceKind               SourceKind     `json:"sourceKind"               bun:"source_kind,type:VARCHAR(20),nullzero"`
	DetectedLanguage         string         `json:"detectedLanguage"         bun:"detected_language,type:VARCHAR(20),nullzero"`
	DetectedDocumentKind     string         `json:"detectedDocumentKind"     bun:"detected_document_kind,type:VARCHAR(100),nullzero"`
	ClassificationConfidence float64        `json:"classificationConfidence" bun:"classification_confidence,type:DOUBLE PRECISION,notnull,default:0"`
	StructuredData           map[string]any `json:"structuredData"           bun:"structured_data,type:JSONB,notnull,default:'{}'"`
	FailureCode              string         `json:"failureCode"              bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage           string         `json:"failureMessage"           bun:"failure_message,type:TEXT,nullzero"`
	SearchVector             string         `json:"-"                        bun:"search_vector,type:TSVECTOR,scanonly"`
	Version                  int64          `json:"version"                  bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt                int64          `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64          `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	LastExtractedAt          *int64         `json:"lastExtractedAt"          bun:"last_extracted_at,type:BIGINT,nullzero"`
	Pages                    []*Page        `json:"pages,omitempty"          bun:"-"`
}

func (c *Content) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("dc_")
		}
		if c.Status == "" {
			c.Status = StatusPending
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *Content) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.DocumentID, validation.Required.Error("Document is required")),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusPending,
				StatusExtracting,
				StatusExtracted,
				StatusIndexed,
				StatusFailed,
			).Error("Status is invalid"),
		),
		validation.Field(&c.SourceKind,
			validation.In(SourceKindNative, SourceKindOCR, SourceKindMixed).
				Error("Source kind is invalid"),
		),
		validation.Field(&c.PageCount,
			validation.Min(0).Error("Page count cannot be negative"),
		),
		// Confidence is reported to the user as a percentage, so a value outside
		// the unit interval would render as nonsense.
		validation.Field(&c.ClassificationConfidence,
			validation.Min(0.0).Error("Confidence must be between 0 and 1"),
			validation.Max(1.0).Error("Confidence must be between 0 and 1"),
		),
		validation.Field(&c.DetectedLanguage,
			validation.Length(0, maxLanguageLength).
				Error("Detected language cannot be longer than 20 characters"),
		),
		validation.Field(&c.DetectedDocumentKind,
			validation.Length(0, maxDetectedKindLength).
				Error("Detected document kind cannot be longer than 100 characters"),
		),
		validation.Field(&c.FailureCode,
			validation.Length(0, maxFailureCodeLength).
				Error("Failure code cannot be longer than 100 characters"),
		),
		validation.Field(&c.FailureMessage,
			validation.When(c.Status == StatusFailed, validation.Required.Error(
				"A failed extraction must record why it failed",
			)),
		),
	))

	for i, page := range c.Pages {
		if page == nil {
			continue
		}
		page.Validate(multiErr.WithIndex("pages", i))
	}
}

const (
	maxLanguageLength     = 20
	maxDetectedKindLength = 100
	maxFailureCodeLength  = 100
)
