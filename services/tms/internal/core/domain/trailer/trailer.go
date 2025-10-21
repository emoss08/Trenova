package trailer

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Trailer)(nil)
	_ domaintypes.PostgresSearchable = (*Trailer)(nil)
	_ domain.Validatable             = (*Trailer)(nil)
	_ framework.TenantedEntity       = (*Trailer)(nil)
)

type Trailer struct {
	bun.BaseModel `bun:"table:trailers,alias:tr" json:"-"`

	ID                      pulid.ID               `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID          pulid.ID               `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID          pulid.ID               `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EquipmentTypeID         pulid.ID               `json:"equipmentTypeId"         bun:"equipment_type_id,type:VARCHAR(100),notnull"`
	EquipmentManufacturerID pulid.ID               `json:"equipmentManufacturerId" bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull"`
	FleetCodeID             *pulid.ID              `json:"fleetCodeId"             bun:"fleet_code_id,type:VARCHAR(100),nullzero"`
	RegistrationStateID     *pulid.ID              `json:"registrationStateId"     bun:"registration_state_id,type:VARCHAR(100),nullzero"`
	Status                  domain.EquipmentStatus `json:"status"                  bun:"status,type:equipment_status_enum,notnull,default:'Available'"`
	Code                    string                 `json:"code"                    bun:"code,type:VARCHAR(50),notnull"`
	Model                   string                 `json:"model"                   bun:"model,type:VARCHAR(50)"`
	Make                    string                 `json:"make"                    bun:"make,type:VARCHAR(50)"`
	LicensePlateNumber      string                 `json:"licensePlateNumber"      bun:"license_plate_number,type:VARCHAR(50)"`
	Vin                     string                 `json:"vin"                     bun:"vin,type:vin_code_optional"`
	RegistrationNumber      string                 `json:"registrationNumber"      bun:"registration_number,type:VARCHAR(50)"`
	SearchVector            string                 `json:"-"                       bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                    string                 `json:"-"                       bun:"rank,type:VARCHAR(100),scanonly"`
	Year                    *int                   `json:"year"                    bun:"year,type:INTEGER,nullzero"`
	MaxLoadWeight           *int                   `json:"maxLoadWeight"           bun:"max_load_weight,type:INTEGER,nullzero"`
	LastInspectionDate      *int64                 `json:"lastInspectionDate"      bun:"last_inspection_date,type:INTEGER,nullzero"`
	RegistrationExpiry      *int64                 `json:"registrationExpiry"      bun:"registration_expiry,type:INTEGER,nullzero"`
	Version                 int64                  `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64                  `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64                  `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          bun:"rel:belongs-to,join:organization_id=id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         bun:"rel:belongs-to,join:equipment_type_id=id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id"`
	RegistrationState     *usstate.UsState                             `json:"state,omitempty"                 bun:"rel:belongs-to,join:registration_state_id=id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             bun:"rel:belongs-to,join:fleet_code_id=id"`
}

func (t *Trailer) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(&t.EquipmentTypeID,
			validation.Required.Error("Equipment Type is required"),
		),
		validation.Field(&t.Make,
			validation.Length(1, 50).Error("Make must be between 1 and 50 characters"),
		),
		validation.Field(&t.Year,
			validation.Min(1900).Error("Year must be between 1900 and 2099"),
			validation.Max(2099).Error("Year must be between 1900 and 2099"),
		),
		validation.Field(&t.Model,
			validation.Length(1, 50).Error("Model must be between 1 and 50 characters"),
		),
		validation.Field(&t.Vin,
			validation.By(domain.ValidateVin),
		),
		validation.Field(&t.EquipmentManufacturerID,
			validation.Required.Error("Equipment Manufacturer is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (t *Trailer) GetID() string {
	return t.ID.String()
}

func (t *Trailer) GetTableName() string {
	return "trailers"
}

func (t *Trailer) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *Trailer) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *Trailer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "tr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "vin", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "license_plate_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
			{
				Name:   "registration_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightD,
			},
		},
	}
}

func (t *Trailer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("tr_")
		}

		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}
