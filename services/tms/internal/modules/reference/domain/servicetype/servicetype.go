package servicetype

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Service level classification
type ServiceType struct {
	keel.Table `keel:"service_types,report=Configuration"`

	ID             pulid.ID           `json:"id"             keel:"id,pk,autoid:st_"`
	BusinessUnitID pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	Status         domaintypes.Status `json:"status"         keel:"status"`
	Code           string             `json:"code"           keel:"code"`
	Description    string             `json:"description"    keel:"description,nullzero"`
	Color          string             `json:"color"          keel:"color,nullzero"`
	Version        int64              `json:"version"        keel:"version,version"`
	CreatedAt      int64              `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64              `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string             `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string             `json:"-"              keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (st *ServiceType) BeforeCreate(_ context.Context) error {
	if st.Status == "" {
		st.Status = domaintypes.StatusActive
	}

	return nil
}

func (st *ServiceType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(st,
		validation.Field(&st.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),
		validation.Field(&st.Color,
			is.HexColor.Error("Color must be a valid hex color. Please try again."),
		),
	))
}

func (st *ServiceType) GetID() pulid.ID {
	return st.ID
}

func (st *ServiceType) GetCreatedAt() int64 {
	return st.CreatedAt
}

func (st *ServiceType) GetTableName() string {
	return "service_types"
}

func (st *ServiceType) GetOrganizationID() pulid.ID {
	return st.OrganizationID
}

func (st *ServiceType) GetBusinessUnitID() pulid.ID {
	return st.BusinessUnitID
}

func (st *ServiceType) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "st",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeText},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
		},
	}
}
