package servicefailure

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*ReasonCode)(nil)
	_ validationframework.TenantedEntity = (*ReasonCode)(nil)
	_ domaintypes.PostgresSearchable     = (*ReasonCode)(nil)
)

type ReasonCode struct {
	bun.BaseModel `bun:"table:service_failure_reason_codes,alias:sfrc" json:"-"`

	ID                   pulid.ID            `json:"id"                    bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID            `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Code                 string              `json:"code"                  bun:"code,type:VARCHAR(64),notnull"`
	Label                string              `json:"label"                 bun:"label,type:VARCHAR(120),notnull"`
	Description          string              `json:"description"           bun:"description,type:TEXT,nullzero"`
	Category             ReasonCategory      `json:"category"              bun:"category,type:service_failure_reason_category_enum,notnull,default:'Carrier'"`
	AppliesTo            ReasonCodeAppliesTo `json:"appliesTo"             bun:"applies_to,type:service_failure_reason_applies_to_enum,notnull,default:'Both'"`
	DefaultStatusCode    string              `json:"defaultStatusCode"     bun:"default_status_code,type:VARCHAR(3),nullzero"`
	DefaultReasonCode    string              `json:"defaultReasonCode"     bun:"default_reason_code,type:VARCHAR(3),nullzero"`
	DefaultExceptionCode string              `json:"defaultExceptionCode"  bun:"default_exception_code,type:VARCHAR(3),nullzero"`
	DefaultNote          string              `json:"defaultNote"           bun:"default_note,type:TEXT,nullzero"`
	Active               bool                `json:"active"                bun:"active,type:BOOLEAN,notnull,default:true"`
	SortOrder            int32               `json:"sortOrder"             bun:"sort_order,type:INTEGER,notnull,default:100"`
	ExternalMap          map[string]any      `json:"externalMap,omitempty" bun:"external_map,type:JSONB,nullzero"`
	ArchivedAt           *int64              `json:"archivedAt"            bun:"archived_at,type:BIGINT,nullzero"`
	ArchivedByID         *pulid.ID           `json:"archivedById"          bun:"archived_by_id,type:VARCHAR(100),nullzero"`
	ActivatedAt          *int64              `json:"activatedAt"           bun:"activated_at,type:BIGINT,nullzero"`
	ActivatedByID        *pulid.ID           `json:"activatedById"         bun:"activated_by_id,type:VARCHAR(100),nullzero"`
	Version              int64               `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64               `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64               `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector         string              `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
	ArchivedBy   *tenant.User         `json:"archivedBy,omitempty"  bun:"rel:belongs-to,join:archived_by_id=id"`
	ActivatedBy  *tenant.User         `json:"activatedBy,omitempty" bun:"rel:belongs-to,join:activated_by_id=id"`
}

func (rc *ReasonCode) Normalize() {
	rc.Code = strings.ToUpper(strings.TrimSpace(rc.Code))
	rc.Label = strings.TrimSpace(rc.Label)
	rc.Description = strings.TrimSpace(rc.Description)
	rc.DefaultStatusCode = strings.ToUpper(strings.TrimSpace(rc.DefaultStatusCode))
	rc.DefaultReasonCode = strings.ToUpper(strings.TrimSpace(rc.DefaultReasonCode))
	rc.DefaultExceptionCode = strings.ToUpper(strings.TrimSpace(rc.DefaultExceptionCode))
	rc.DefaultNote = strings.TrimSpace(rc.DefaultNote)
}

func (rc *ReasonCode) Validate(multiErr *errortypes.MultiError) {
	rc.Normalize()
	err := validation.ValidateStruct(rc,
		validation.Field(&rc.OrganizationID, validation.Required.Error("Organization ID is required")),
		validation.Field(&rc.BusinessUnitID, validation.Required.Error("Business unit ID is required")),
		validation.Field(
			&rc.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 64).Error("Code must be between 1 and 64 characters"),
		),
		validation.Field(
			&rc.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 120).Error("Label must be between 1 and 120 characters"),
		),
		validation.Field(
			&rc.Category,
			validation.Required.Error("Category is required"),
			validation.By(func(value any) error {
				category, _ := value.(ReasonCategory)
				if !category.IsValid() {
					return errors.New("category is invalid")
				}
				return nil
			}),
		),
		validation.Field(
			&rc.AppliesTo,
			validation.Required.Error("Applies to is required"),
			validation.By(func(value any) error {
				appliesTo, _ := value.(ReasonCodeAppliesTo)
				if !appliesTo.IsValid() {
					return errors.New("applies to is invalid")
				}
				return nil
			}),
		),
		validation.Field(
			&rc.DefaultStatusCode,
			validation.Length(0, 3).Error("Default status code must be at most 3 characters"),
		),
		validation.Field(
			&rc.DefaultReasonCode,
			validation.Length(0, 3).Error("Default reason code must be at most 3 characters"),
		),
		validation.Field(
			&rc.DefaultExceptionCode,
			validation.Length(0, 3).Error("Default exception code must be at most 3 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (rc *ReasonCode) GetID() pulid.ID {
	return rc.ID
}

func (rc *ReasonCode) GetTableName() string {
	return "service_failure_reason_codes"
}

func (rc *ReasonCode) GetOrganizationID() pulid.ID {
	return rc.OrganizationID
}

func (rc *ReasonCode) GetBusinessUnitID() pulid.ID {
	return rc.BusinessUnitID
}

func (rc *ReasonCode) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sfrc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "label", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "category", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "applies_to", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{Name: "description", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (rc *ReasonCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	rc.Normalize()

	switch query.(type) {
	case *bun.InsertQuery:
		if rc.ID.IsNil() {
			rc.ID = pulid.MustNew("sfrc_")
		}
		if rc.Category == "" {
			rc.Category = ReasonCategoryCarrier
		}
		if rc.AppliesTo == "" {
			rc.AppliesTo = ReasonCodeAppliesToBoth
		}
		rc.CreatedAt = now
	case *bun.UpdateQuery:
		rc.UpdatedAt = now
	}

	return nil
}
