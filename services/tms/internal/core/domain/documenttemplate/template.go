package documenttemplate

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*DocumentTemplate)(nil)
	_ domaintypes.PostgresSearchable = (*DocumentTemplate)(nil)
	_ domain.Validatable             = (*DocumentTemplate)(nil)
	_ framework.TenantedEntity       = (*DocumentTemplate)(nil)
)

type DocumentTemplate struct {
	bun.BaseModel `bun:"table:document_templates,alias:dtpl" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Code           string         `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(200),notnull"`
	Description    string         `json:"description"    bun:"description,type:TEXT,nullzero"`
	DocumentTypeID pulid.ID       `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),notnull"`
	HTMLContent    string         `json:"htmlContent"    bun:"html_content,type:TEXT,notnull"`
	CSSContent     string         `json:"cssContent"     bun:"css_content,type:TEXT,nullzero"`
	HeaderHTML     string         `json:"headerHtml"     bun:"header_html,type:TEXT,nullzero"`
	FooterHTML     string         `json:"footerHtml"     bun:"footer_html,type:TEXT,nullzero"`
	PageSize       PageSize       `json:"pageSize"       bun:"page_size,type:VARCHAR(20),notnull,default:'Letter'"`
	Orientation    Orientation    `json:"orientation"    bun:"orientation,type:VARCHAR(20),notnull,default:'Portrait'"`
	MarginTop      int32          `json:"marginTop"      bun:"margin_top,type:INT,notnull,default:20"`
	MarginBottom   int32          `json:"marginBottom"   bun:"margin_bottom,type:INT,notnull,default:20"`
	MarginLeft     int32          `json:"marginLeft"     bun:"margin_left,type:INT,notnull,default:20"`
	MarginRight    int32          `json:"marginRight"    bun:"margin_right,type:INT,notnull,default:20"`
	Status         TemplateStatus `json:"status"         bun:"status,type:VARCHAR(20),notnull,default:'Draft'"`
	IsDefault      bool           `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull,default:false"`
	IsSystem       bool           `json:"isSystem"       bun:"is_system,type:BOOLEAN,notnull,default:false"`
	SearchVector   string         `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string         `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CreatedByID    *pulid.ID      `json:"createdById"    bun:"created_by_id,type:VARCHAR(100),nullzero"`
	UpdatedByID    *pulid.ID      `json:"updatedById"    bun:"updated_by_id,type:VARCHAR(100),nullzero"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit       `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization       `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	DocumentType *documenttype.DocumentType `json:"documentType,omitempty" bun:"rel:belongs-to,join:document_type_id=id,join:business_unit_id=business_unit_id,join:organization_id=organization_id"`
	CreatedBy    *tenant.User               `json:"createdBy,omitempty"    bun:"rel:belongs-to,join:created_by_id=id"`
	UpdatedBy    *tenant.User               `json:"updatedBy,omitempty"    bun:"rel:belongs-to,join:updated_by_id=id"`
}

func (dt *DocumentTemplate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dt,
		validation.Field(
			&dt.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(
			&dt.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 200).Error("Name must be between 1 and 200 characters"),
		),
		validation.Field(
			&dt.DocumentTypeID,
			validation.Required.Error("Document type is required"),
		),
		validation.Field(
			&dt.HTMLContent,
			validation.Required.Error("HTML content is required"),
		),
		validation.Field(
			&dt.PageSize,
			validation.Required.Error("Page size is required"),
			validation.In(PageSizeLetter, PageSizeA4, PageSizeLegal).
				Error("Page size must be Letter, A4, or Legal"),
		),
		validation.Field(
			&dt.Orientation,
			validation.Required.Error("Orientation is required"),
			validation.In(OrientationPortrait, OrientationLandscape).
				Error("Orientation must be Portrait or Landscape"),
		),
		validation.Field(
			&dt.Status,
			validation.Required.Error("Status is required"),
			validation.In(TemplateStatusDraft, TemplateStatusActive, TemplateStatusArchived).
				Error("Status must be Draft, Active, or Archived"),
		),
		validation.Field(
			&dt.MarginTop,
			validation.Min(0).Error("Margin top must be non-negative"),
			validation.Max(100).Error("Margin top must be at most 100mm"),
		),
		validation.Field(
			&dt.MarginBottom,
			validation.Min(0).Error("Margin bottom must be non-negative"),
			validation.Max(100).Error("Margin bottom must be at most 100mm"),
		),
		validation.Field(
			&dt.MarginLeft,
			validation.Min(0).Error("Margin left must be non-negative"),
			validation.Max(100).Error("Margin left must be at most 100mm"),
		),
		validation.Field(
			&dt.MarginRight,
			validation.Min(0).Error("Margin right must be non-negative"),
			validation.Max(100).Error("Margin right must be at most 100mm"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dt *DocumentTemplate) GetID() string {
	return dt.ID.String()
}

func (dt *DocumentTemplate) GetTableName() string {
	return "document_templates"
}

func (dt *DocumentTemplate) GetOrganizationID() pulid.ID {
	return dt.OrganizationID
}

func (dt *DocumentTemplate) GetBusinessUnitID() pulid.ID {
	return dt.BusinessUnitID
}

func (dt *DocumentTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "dtpl",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (dt *DocumentTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dt.ID.IsNil() {
			dt.ID = pulid.MustNew("dtpl_")
		}
		dt.CreatedAt = now
	case *bun.UpdateQuery:
		dt.UpdatedAt = now
	}

	return nil
}

func (dt *DocumentTemplate) CanBeDeleted() bool {
	return !dt.IsSystem
}

func (dt *DocumentTemplate) CanBeEdited() bool {
	return dt.Status != TemplateStatusArchived
}
