package servicefailure

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*ReasonCode)(nil)
	_ domaintypes.PostgresSearchable     = (*ReasonCode)(nil)
	_ pagination.CursorEntity            = (*ReasonCode)(nil)
)

type ReasonCode struct {
	keel.Table `keel:"service_failure_reason_codes"`

	ID                   pulid.ID            `json:"id" keel:"id,pk,autoid:sfrc_"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	Code                 string              `json:"code" keel:"code"`
	Label                string              `json:"label" keel:"label"`
	Description          string              `json:"description" keel:"description,nullzero"`
	Category             ReasonCategory      `json:"category" keel:"category"`
	AppliesTo            ReasonCodeAppliesTo `json:"appliesTo" keel:"applies_to"`
	DefaultStatusCode    string              `json:"defaultStatusCode" keel:"default_status_code,nullzero"`
	DefaultReasonCode    string              `json:"defaultReasonCode" keel:"default_reason_code,nullzero"`
	DefaultExceptionCode string              `json:"defaultExceptionCode" keel:"default_exception_code,nullzero"`
	DefaultNote          string              `json:"defaultNote" keel:"default_note,nullzero"`
	Active               bool                `json:"active" keel:"active"`
	SortOrder            int32               `json:"sortOrder" keel:"sort_order"`
	ExternalMap          map[string]any      `json:"externalMap,omitempty" keel:"external_map,jsonb,nullzero"`
	ArchivedAt           *int64              `json:"archivedAt" keel:"archived_at"`
	ArchivedByID         *pulid.ID           `json:"archivedById" keel:"archived_by_id"`
	ActivatedAt          *int64              `json:"activatedAt" keel:"activated_at"`
	ActivatedByID        *pulid.ID           `json:"activatedById" keel:"activated_by_id"`
	Version              int64               `json:"version" keel:"version,version"`
	CreatedAt            int64               `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt            int64               `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector         string              `json:"-" keel:"search_vector,tsvector,readonly,noselect"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	ArchivedBy   *tenant.User         `json:"archivedBy,omitempty" keel:"belongs:archived_by_id"`
	ActivatedBy  *tenant.User         `json:"activatedBy,omitempty" keel:"belongs:activated_by_id"`
}

func (rc *ReasonCode) BeforeUpdate(_ context.Context) error {
	rc.Normalize()

	return nil
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
	multiErr.AddOzzoError(validation.ValidateStruct(
		rc,
		validation.Field(
			&rc.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&rc.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
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
			domainvalidation.ValidEnum[ReasonCategory]("category is invalid"),
		),
		validation.Field(
			&rc.AppliesTo,
			validation.Required.Error("Applies to is required"),
			domainvalidation.ValidEnum[ReasonCodeAppliesTo]("applies to is invalid"),
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
	))
}

func (rc *ReasonCode) GetID() pulid.ID {
	return rc.ID
}

func (rc *ReasonCode) GetCreatedAt() int64 {
	return rc.CreatedAt
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
			{
				Name:   "applies_to",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (rc *ReasonCode) BeforeCreate(ctx context.Context) error {
	if rc.Category == "" {
		rc.Category = ReasonCategoryCarrier
	}
	if rc.AppliesTo == "" {
		rc.AppliesTo = ReasonCodeAppliesToBoth
	}
	if rc.SortOrder == 0 {
		rc.SortOrder = 100
	}
	rc.Normalize()

	return nil
}
