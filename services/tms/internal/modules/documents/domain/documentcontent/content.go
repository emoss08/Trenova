package documentcontent

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"document_contents"`

	ID                       pulid.ID       `json:"id"                       keel:"id,pk,autoid:dc_"`
	DocumentID               pulid.ID       `json:"documentId"               keel:"document_id"`
	OrganizationID           pulid.ID       `json:"organizationId"           keel:"organization_id,pk,tenant"`
	BusinessUnitID           pulid.ID       `json:"businessUnitId"           keel:"business_unit_id,pk,tenant"`
	Status                   Status         `json:"status"                   keel:"status"`
	ContentText              string         `json:"contentText"              keel:"content_text,nullzero"`
	PageCount                int            `json:"pageCount"                keel:"page_count"`
	SourceKind               SourceKind     `json:"sourceKind"               keel:"source_kind,nullzero"`
	DetectedLanguage         string         `json:"detectedLanguage"         keel:"detected_language,nullzero"`
	DetectedDocumentKind     string         `json:"detectedDocumentKind"     keel:"detected_document_kind,nullzero"`
	ClassificationConfidence float64        `json:"classificationConfidence" keel:"classification_confidence"`
	StructuredData           map[string]any `json:"structuredData"           keel:"structured_data,jsonb"`
	FailureCode              string         `json:"failureCode"              keel:"failure_code,nullzero"`
	FailureMessage           string         `json:"failureMessage"           keel:"failure_message,nullzero"`
	SearchVector             string         `json:"-"                        keel:"search_vector,tsvector,readonly,noselect"`
	Version                  int64          `json:"version"                  keel:"version,version"`
	CreatedAt                int64          `json:"createdAt"                keel:"created_at,autocreate"`
	UpdatedAt                int64          `json:"updatedAt"                keel:"updated_at,autoupdate"`
	LastExtractedAt          *int64         `json:"lastExtractedAt"          keel:"last_extracted_at"`
	ConverterVersion         string         `json:"converterVersion"         keel:"converter_version,nullzero"`
	RawObjectKey             string         `json:"-"                        keel:"raw_object_key,nullzero"`
	Pages                    []*Page        `json:"pages,omitempty"`
}

func (c *Content) BeforeCreate(ctx context.Context) error {
	if c.Status == "" {
		c.Status = StatusPending
	}
	c.stripNUL()

	return nil
}

func (c *Content) BeforeUpdate(_ context.Context) error {
	c.stripNUL()
	c.stripNUL()

	return nil
}

func (c *Content) stripNUL() {
	c.ContentText = stringutils.StripNUL(c.ContentText)
	c.FailureMessage = stringutils.StripNUL(c.FailureMessage)
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
