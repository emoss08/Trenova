package documentcontent

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Page struct {
	bun.BaseModel `bun:"table:document_content_pages,alias:dcp" json:"-"`

	ID                   pulid.ID       `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	DocumentContentID    pulid.ID       `json:"documentContentId"    bun:"document_content_id,type:VARCHAR(100),notnull"`
	DocumentID           pulid.ID       `json:"documentId"           bun:"document_id,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID       `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID       pulid.ID       `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	PageNumber           int            `json:"pageNumber"           bun:"page_number,type:INTEGER,notnull"`
	SourceKind           SourceKind     `json:"sourceKind"           bun:"source_kind,type:VARCHAR(20),notnull"`
	ExtractedText        string         `json:"extractedText"        bun:"extracted_text,type:TEXT,nullzero"`
	OCRConfidence        float64        `json:"ocrConfidence"        bun:"ocr_confidence,type:DOUBLE PRECISION,notnull,default:0"`
	PreprocessingApplied bool           `json:"preprocessingApplied" bun:"preprocessing_applied,type:BOOLEAN,notnull,default:false"`
	Width                int            `json:"width"                bun:"width,type:INTEGER,notnull,default:0"`
	Height               int            `json:"height"               bun:"height,type:INTEGER,notnull,default:0"`
	Metadata             map[string]any `json:"metadata"             bun:"metadata,type:JSONB,notnull,default:'{}'"`
	Version              int64          `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64          `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64          `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *Page) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("dcp_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
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
