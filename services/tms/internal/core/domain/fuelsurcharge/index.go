package fuelsurcharge

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*FuelIndex)(nil)
	_ validationframework.TenantedEntity = (*FuelIndex)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelIndex)(nil)
)

type FuelIndex struct {
	bun.BaseModel             `bun:"table:fuel_indices,alias:fidx" json:"-"`
	pagination.CursorValueSet `bun:",embed"                        json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string      `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Code           string      `json:"code"           bun:"code,type:VARCHAR(50),notnull"`
	Description    string      `json:"description"    bun:"description,type:TEXT,nullzero"`
	Source         IndexSource `json:"source"         bun:"source,type:fuel_index_source_enum,notnull"`
	FuelType       FuelType    `json:"fuelType"       bun:"fuel_type,type:fuel_type_enum,notnull,default:'Diesel'"`
	Region         string      `json:"region"         bun:"region,type:VARCHAR(100),nullzero"`
	EIASeriesID    string      `json:"eiaSeriesId"    bun:"eia_series_id,type:VARCHAR(64),nullzero"`
	Currency       string      `json:"currency"       bun:"currency,type:VARCHAR(3),notnull,default:'USD'"`
	IsActive       bool        `json:"isActive"       bun:"is_active,type:BOOLEAN,notnull,default:true"`
	Version        int64       `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64       `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector   string      `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string      `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (fi *FuelIndex) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(fi,
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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

func (fi *FuelIndex) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if fi.ID.IsNil() {
			fi.ID = pulid.MustNew("fidx_")
		}
		if fi.FuelType == "" {
			fi.FuelType = FuelTypeDiesel
		}
		fi.CreatedAt = now
		fi.UpdatedAt = now
	case *bun.UpdateQuery:
		fi.UpdatedAt = now
	}

	return nil
}
