package documentcontent

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Page struct {
	keel.Table `keel:"document_content_pages"`

	ID                   pulid.ID       `json:"id"                   keel:"id,pk,autoid:dcp_"`
	DocumentContentID    pulid.ID       `json:"documentContentId"    keel:"document_content_id"`
	DocumentID           pulid.ID       `json:"documentId"           keel:"document_id"`
	OrganizationID       pulid.ID       `json:"organizationId"       keel:"organization_id,tenant"`
	BusinessUnitID       pulid.ID       `json:"businessUnitId"       keel:"business_unit_id,tenant"`
	PageNumber           int            `json:"pageNumber"           keel:"page_number"`
	SourceKind           SourceKind     `json:"sourceKind"           keel:"source_kind"`
	ExtractedText        string         `json:"extractedText"        keel:"extracted_text,nullzero"`
	Markdown             string         `json:"markdown"             keel:"markdown,nullzero"`
	OCRConfidence        float64        `json:"ocrConfidence"        keel:"ocr_confidence"`
	PreprocessingApplied bool           `json:"preprocessingApplied" keel:"preprocessing_applied"`
	Width                int            `json:"width"                keel:"width"`
	Height               int            `json:"height"               keel:"height"`
	Metadata             map[string]any `json:"metadata"             keel:"metadata,jsonb"`
	Version              int64          `json:"version"              keel:"version,version"`
	CreatedAt            int64          `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64          `json:"updatedAt"            keel:"updated_at,autoupdate"`
}

func (p *Page) BeforeUpdate(_ context.Context) error {
	p.stripNUL()

	return nil
}

func (p *Page) BeforeCreate(_ context.Context) error {
	p.stripNUL()

	return nil
}

func (p *Page) BeforeSave(_ context.Context) error {
	p.stripNUL()
	return nil
}

func (p *Page) stripNUL() {
	p.ExtractedText = stringutils.StripNUL(p.ExtractedText)
	p.Markdown = stringutils.StripNUL(p.Markdown)
}

func (p *Page) Validate(multiErr *errortypes.MultiError) {
	// Tenancy and the owning content are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.PageNumber,
			validation.Required.Error("Page number is required"),
			validation.Min(1).Error("Page number must be at least one"),
		),
		validation.Field(&p.SourceKind,
			validation.Required.Error("Source kind is required"),
			validation.In(SourceKindNative, SourceKindOCR, SourceKindMixed).
				Error("Source kind is invalid"),
		),
		validation.Field(&p.OCRConfidence,
			validation.Min(0.0).Error("OCR confidence must be between 0 and 1"),
			validation.Max(1.0).Error("OCR confidence must be between 0 and 1"),
		),
		validation.Field(&p.Width, validation.Min(0).Error("Width cannot be negative")),
		validation.Field(&p.Height, validation.Min(0).Error("Height cannot be negative")),
	))
}

func (p *Page) ModelText() string {
	if strings.TrimSpace(p.Markdown) != "" {
		return p.Markdown
	}

	return p.ExtractedText
}
