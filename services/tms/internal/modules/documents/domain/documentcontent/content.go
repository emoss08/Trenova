package documentcontent

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
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

func (s Status) Settled() bool {
	return s == StatusExtracted || s == StatusIndexed || s == StatusFailed
}

type Content struct {
	bun.BaseModel `bun:"table:document_contents,alias:dc" json:"-"`

	ID                       pulid.ID       `json:"id"                       bun:"id,type:VARCHAR(100),pk,notnull"`
	DocumentID               pulid.ID       `json:"documentId"               bun:"document_id,type:VARCHAR(100),notnull"`
	OrganizationID           pulid.ID       `json:"organizationId"           bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID           pulid.ID       `json:"businessUnitId"           bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	Status                   Status         `json:"status"                   bun:"status,type:document_content_status_enum,notnull,default:'Pending'"`
	ContentText              string         `json:"contentText"              bun:"content_text,type:TEXT,nullzero"`
	PageCount                int            `json:"pageCount"                bun:"page_count,type:INTEGER,notnull"`
	SourceKind               SourceKind     `json:"sourceKind"               bun:"source_kind,type:VARCHAR(20),nullzero"`
	DetectedLanguage         string         `json:"detectedLanguage"         bun:"detected_language,type:VARCHAR(20),nullzero"`
	DetectedDocumentKind     string         `json:"detectedDocumentKind"     bun:"detected_document_kind,type:VARCHAR(100),nullzero"`
	ClassificationConfidence float64        `json:"classificationConfidence" bun:"classification_confidence,type:DOUBLE PRECISION,notnull"`
	StructuredData           map[string]any `json:"structuredData"           bun:"structured_data,type:JSONB,notnull,default:'{}'"`
	FailureCode              string         `json:"failureCode"              bun:"failure_code,type:VARCHAR(100),nullzero"`
	FailureMessage           string         `json:"failureMessage"           bun:"failure_message,type:TEXT,nullzero"`
	SearchVector             string         `json:"-"                        bun:"search_vector,type:TSVECTOR,scanonly"`
	Version                  int64          `json:"version"                  bun:"version,type:BIGINT,notnull"`
	CreatedAt                int64          `json:"createdAt"                bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                int64          `json:"updatedAt"                bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	LastExtractedAt          *int64         `json:"lastExtractedAt"          bun:"last_extracted_at,type:BIGINT,nullzero"`
	ConverterVersion         string         `json:"converterVersion"         bun:"converter_version,type:VARCHAR(40),nullzero"`
	RawObjectKey             string         `json:"-"                        bun:"raw_object_key,type:TEXT,nullzero"`
	Pages                    []*Page        `json:"pages,omitempty"          bun:"-"`
}

func (c *Content) BeforeAppendModel(_ context.Context, query bun.Query) error {
	c.ContentText = stringutils.StripNUL(c.ContentText)
	c.FailureMessage = stringutils.StripNUL(c.FailureMessage)

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
		validation.Field(&c.ConverterVersion,
			validation.Length(0, maxConverterVersionLength).
				Error("Converter version cannot be longer than 40 characters"),
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

	maxConverterVersionLength = 40
)

// poorReadingConfidence is the average OCR confidence under which a scan is
// taken to be mostly unreadable: blurred, faint or photographed at an angle.
const poorReadingConfidence = 0.5

// PoorlyRead reports whether reading finished but made out little of the
// file: the pages read by OCR were read with low confidence on average, as a
// blurred or skewed photo is. Text a file carried natively is always legible,
// and a reading that failed says nothing about the file itself.
func (c *Content) PoorlyRead() bool {
	if c == nil {
		return false
	}
	if c.Status != StatusExtracted && c.Status != StatusIndexed {
		return false
	}

	var total float64
	var scanned int
	for _, page := range c.Pages {
		if page == nil || page.SourceKind != SourceKindOCR {
			continue
		}
		total += page.OCRConfidence
		scanned++
	}

	return scanned > 0 && total/float64(scanned) < poorReadingConfidence
}

func (c *Content) PagedText() string {
	pages := make([]*Page, 0, len(c.Pages))
	for _, page := range c.Pages {
		if page != nil && strings.TrimSpace(page.ExtractedText) != "" {
			pages = append(pages, page)
		}
	}
	if len(pages) < 2 {
		return strings.TrimSpace(c.ContentText)
	}
	slices.SortFunc(pages, func(a, b *Page) int { return cmp.Compare(a.PageNumber, b.PageNumber) })

	var b strings.Builder
	for _, page := range pages {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(
			&b,
			"--- page %d ---\n%s",
			page.PageNumber,
			strings.TrimSpace(page.ExtractedText),
		)
	}

	return b.String()
}

func RawObjectKey(organizationID, documentID pulid.ID) string {
	return organizationID.String() + "/document-extraction/" + documentID.String() + "/docling.json.gz"
}
