package tractor

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook         = (*Tractor)(nil)
	_ domaintypes.PostgresSearchable    = (*Tractor)(nil)
	_ customfield.CustomFieldsSupporter = (*Tractor)(nil)
)

type Tractor struct {
	bun.BaseModel `bun:"table:tractors,alias:trac" json:"-"`

	ID                      pulid.ID                    `json:"id"                      bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID          pulid.ID                    `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID          pulid.ID                    `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	PrimaryWorkerID         pulid.ID                    `json:"primaryWorkerId"         bun:"primary_worker_id,type:VARCHAR(100),notnull"`
	EquipmentTypeID         pulid.ID                    `json:"equipmentTypeId"         bun:"equipment_type_id,type:VARCHAR(100),notnull"`
	EquipmentManufacturerID pulid.ID                    `json:"equipmentManufacturerId" bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull"`
	StateID                 pulid.ID                    `json:"stateId"                 bun:"state_id,type:VARCHAR(100),nullzero"`
	FleetCodeID             pulid.ID                    `json:"fleetCodeId"             bun:"fleet_code_id,type:VARCHAR(100),nullzero"`
	SecondaryWorkerID       pulid.ID                    `json:"secondaryWorkerId"       bun:"secondary_worker_id,type:VARCHAR(100),nullzero"`
	Status                  domaintypes.EquipmentStatus `json:"status"                  bun:"status,type:equipment_status_enum,notnull,default:'Available'"`
	Code                    string                      `json:"code"                    bun:"code,type:VARCHAR(50),notnull"`
	Model                   string                      `json:"model"                   bun:"model,type:VARCHAR(50),nullzero"`
	Make                    string                      `json:"make"                    bun:"make,type:VARCHAR(50),nullzero"`
	Year                    *int                        `json:"year"                    bun:"year,type:INT,nullzero"`
	LicensePlateNumber      string                      `json:"licensePlateNumber"      bun:"license_plate_number,type:VARCHAR(50),nullzero"`
	RegistrationNumber      string                      `json:"registrationNumber"      bun:"registration_number,type:VARCHAR(50),nullzero"`
	RegistrationExpiry      *int64                      `json:"registrationExpiry"      bun:"registration_expiry,type:BIGINT,nullzero"`
	Vin                     string                      `json:"vin"                     bun:"vin,type:vin_code_optional,nullzero"`
	Version                 int64                       `json:"version"                 bun:"version,type:BIGINT"`
	CreatedAt               int64                       `json:"createdAt"               bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64                       `json:"updatedAt"               bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CustomFields map[string]any `json:"customFields,omitempty" bun:"-"`

	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          bun:"rel:belongs-to,join:organization_id=id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         bun:"rel:belongs-to,join:equipment_type_id=id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             bun:"rel:belongs-to,join:fleet_code_id=id"`
	State                 *usstate.UsState                             `json:"state,omitempty"                 bun:"rel:belongs-to,join:state_id=id"`
	PrimaryWorker         *worker.Worker                               `json:"primaryWorker,omitempty"         bun:"rel:belongs-to,join:primary_worker_id=id"`
	SecondaryWorker       *worker.Worker                               `json:"secondaryWorker,omitempty"       bun:"rel:belongs-to,join:secondary_worker_id=id"`
}

func (t *Tractor) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		t,
		validation.Field(&t.Code, validation.Required),
		validation.Field(
			&t.Code,
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(
			&t.Make,
			validation.Length(0, 50).Error("Make must be between 0 and 50 characters"),
		),
		validation.Field(
			&t.Model,
			validation.Length(0, 50).Error("Model must be between 0 and 50 characters"),
		),
		validation.Field(
			&t.Year,
			validation.Min(1900).Error("Year must be between 1900 and 2099"),
			validation.Max(2099).Error("Year must be between 1900 and 2099"),
		),
		validation.Field(
			&t.EquipmentTypeID,
			validation.Required.Error("Equipment Type is required"),
		),
		validation.Field(
			&t.EquipmentManufacturerID,
			validation.Required.Error("Equipment Manufacturer is required"),
		),
		validation.Field(
			&t.PrimaryWorkerID,
			validation.Required.Error("Primary Worker is required"),
		),
		validation.Field(
			&t.Vin,
			validation.By(domaintypes.ValidateVin),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (t *Tractor) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("trac_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

func (t *Tractor) GetID() pulid.ID {
	return t.ID
}

func (t *Tractor) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *Tractor) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *Tractor) GetTableName() string {
	return "tractors"
}

func (t *Tractor) GetResourceType() string {
	return "tractor"
}

func (t *Tractor) GetResourceID() string {
	return t.ID.String()
}

func (t *Tractor) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "trac",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "vin", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "license_plate_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (t *Tractor) SetCustomFields(fields map[string]any) {
	t.CustomFields = fields
}

func (t *Tractor) GetCustomFields() map[string]any {
	return t.CustomFields
}
