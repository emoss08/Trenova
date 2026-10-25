package fuelsurcharge

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*FuelIndex)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelIndex)(nil)
)

type FuelIndex struct {
	keel.Table `keel:"fuel_indices"`

	ID             pulid.ID    `json:"id"             keel:"id,pk,autoid:fidx_"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID    `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name           string      `json:"name"           keel:"name"`
	Code           string      `json:"code"           keel:"code"`
	Description    string      `json:"description"    keel:"description,nullzero"`
	Source         IndexSource `json:"source"         keel:"source"`
	FuelType       FuelType    `json:"fuelType"       keel:"fuel_type"`
	Region         string      `json:"region"         keel:"region,nullzero"`
	EIASeriesID    string      `json:"eiaSeriesId"    keel:"eia_series_id,nullzero"`
	Currency       string      `json:"currency"       keel:"currency"`
	IsActive       bool        `json:"isActive"       keel:"is_active"`
	Version        int64       `json:"version"        keel:"version,version"`
	CreatedAt      int64       `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64       `json:"updatedAt"      keel:"updated_at,autoupdate"`
	SearchVector   string      `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string      `json:"-"              keel:"rank,computed"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (fi *FuelIndex) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(fi,
		validation.Field(&fi.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100),
		),
		validation.Field(&fi.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50),
		),
		validation.Field(&fi.Source,
			validation.Required.Error("Source is required"),
			validation.In(IndexSourceEIA, IndexSourceCustom).Error("Source is invalid"),
		),
		validation.Field(&fi.FuelType,
			validation.Required.Error("Fuel type is required"),
			validation.In(FuelTypeDiesel, FuelTypeGasoline).Error("Fuel type is invalid"),
		),
		validation.Field(&fi.Region,
			validation.Length(0, 100),
		),
		validation.Field(&fi.EIASeriesID,
			validation.When(fi.Source == IndexSourceEIA,
				validation.Required.Error("EIA series is required for EIA indices"),
				validation.By(validateEIASeriesID),
			),
			validation.When(fi.Source == IndexSourceCustom,
				validation.Empty.Error("EIA series must be empty for custom indices"),
			),
		),
		validation.Field(&fi.Currency,
			validation.Required.Error("Currency is required"),
			validation.Length(3, 3).Error("Currency must be 3 characters"),
		),
	))
}

func validateEIASeriesID(value any) error {
	seriesID, ok := value.(string)
	if !ok || seriesID == "" {
		return nil
	}
	if _, found := EIASeriesByID(seriesID); !found {
		return errors.New("unknown EIA series")
	}
	return nil
}

func (fi *FuelIndex) GetID() pulid.ID {
	return fi.ID
}

func (fi *FuelIndex) GetCreatedAt() int64 {
	return fi.CreatedAt
}

func (fi *FuelIndex) GetOrganizationID() pulid.ID {
	return fi.OrganizationID
}

func (fi *FuelIndex) GetBusinessUnitID() pulid.ID {
	return fi.BusinessUnitID
}

func (fi *FuelIndex) GetTableName() string {
	return "fuel_indices"
}

func (fi *FuelIndex) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fidx",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "code", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
		},
	}
}

func (fi *FuelIndex) BeforeCreate(ctx context.Context) error {
	if fi.FuelType == "" {
		fi.FuelType = FuelTypeDiesel
	}
	if fi.Currency == "" {
		fi.Currency = "USD"
	}

	return nil
}

func (fi *FuelIndex) BeforeSave(_ context.Context) error {
	if fi.FuelType == "" {
		fi.FuelType = FuelTypeDiesel
	}
	return nil
}
