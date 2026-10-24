package tractor

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Tractor)(nil)
	_ domaintypes.PostgresSearchable     = (*Tractor)(nil)
	_ customfield.CustomFieldsSupporter  = (*Tractor)(nil)
	_ pagination.CursorEntity            = (*Tractor)(nil)
	_ validationframework.TenantedEntity = (*Tractor)(nil)
)

type Tractor struct {
	bun.BaseModel `bun:"table:tractors,alias:trac" json:"-"`
	keel.Table    `                                         keel:"tractors"`

	ID                         pulid.ID                    `json:"id"                         bun:"id,type:VARCHAR(100),pk,notnull"                                                      keel:"id,pk,autoid:trac_"`
	BusinessUnitID             pulid.ID                    `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID             pulid.ID                    `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),notnull,pk"                                         keel:"organization_id,pk,tenant"`
	PrimaryWorkerID            pulid.ID                    `json:"primaryWorkerId"            bun:"primary_worker_id,type:VARCHAR(100),notnull"                                          keel:"primary_worker_id"`
	EquipmentTypeID            pulid.ID                    `json:"equipmentTypeId"            bun:"equipment_type_id,type:VARCHAR(100),notnull"                                          keel:"equipment_type_id"`
	EquipmentManufacturerID    pulid.ID                    `json:"equipmentManufacturerId"    bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull"                                  keel:"equipment_manufacturer_id"`
	StateID                    pulid.ID                    `json:"stateId"                    bun:"state_id,type:VARCHAR(100),nullzero"                                                  keel:"state_id,nullzero"`
	FleetCodeID                pulid.ID                    `json:"fleetCodeId"                bun:"fleet_code_id,type:VARCHAR(100),nullzero"                                             keel:"fleet_code_id,nullzero"`
	SecondaryWorkerID          pulid.ID                    `json:"secondaryWorkerId"          bun:"secondary_worker_id,type:VARCHAR(100),nullzero"                                       keel:"secondary_worker_id,nullzero"`
	Status                     domaintypes.EquipmentStatus `json:"status"                     bun:"status,type:equipment_status_enum,notnull,default:'Available'"                        keel:"status"`
	Code                       string                      `json:"code"                       bun:"code,type:VARCHAR(50),notnull"                                                        keel:"code"`
	Model                      string                      `json:"model"                      bun:"model,type:VARCHAR(50),nullzero"                                                      keel:"model,nullzero"`
	Make                       string                      `json:"make"                       bun:"make,type:VARCHAR(50),nullzero"                                                       keel:"make,nullzero"`
	Year                       *int                        `json:"year"                       bun:"year,type:INT,nullzero"                                                               keel:"year"`
	LicensePlateNumber         string                      `json:"licensePlateNumber"         bun:"license_plate_number,type:VARCHAR(50),nullzero"                                       keel:"license_plate_number,nullzero"`
	RegistrationNumber         string                      `json:"registrationNumber"         bun:"registration_number,type:VARCHAR(50),nullzero"                                        keel:"registration_number,nullzero"`
	RegistrationExpiry         *int64                      `json:"registrationExpiry"         bun:"registration_expiry,type:BIGINT,nullzero"                                             keel:"registration_expiry"`
	FuelType                   domaintypes.IFTAFuelType    `json:"fuelType"                   bun:"fuel_type,type:ifta_fuel_type_enum,notnull"                                           keel:"fuel_type"`
	IFTAQualified              bool                        `json:"iftaQualified"              bun:"ifta_qualified,type:BOOLEAN,notnull"                                                  keel:"ifta_qualified"`
	Vin                        string                      `json:"vin"                        bun:"vin,type:vin_code_optional,nullzero"                                                  keel:"vin,nullzero"`
	ExternalID                 string                      `json:"externalId"                 bun:"external_id,type:TEXT,nullzero"                                                       keel:"external_id,nullzero"`
	OwnershipType              domaintypes.OwnershipType   `json:"ownershipType"              bun:"ownership_type,type:VARCHAR(50),notnull,default:'CompanyOwned'"                       keel:"ownership_type"`
	OwnerWorkerID              *pulid.ID                   `json:"ownerWorkerId"              bun:"owner_worker_id,type:VARCHAR(100),nullzero"                                           keel:"owner_worker_id"`
	LessorName                 string                      `json:"lessorName"                 bun:"lessor_name,type:VARCHAR(150),nullzero"                                               keel:"lessor_name,nullzero"`
	LeaseReference             string                      `json:"leaseReference"             bun:"lease_reference,type:VARCHAR(100),nullzero"                                           keel:"lease_reference,nullzero"`
	LeaseEndDate               *int64                      `json:"leaseEndDate"               bun:"lease_end_date,type:BIGINT,nullzero"                                                  keel:"lease_end_date"`
	LastKnownLocationID        pulid.ID                    `json:"lastKnownLocationId"        bun:"last_known_location_id,type:VARCHAR(100),scanonly"                                    keel:"last_known_location_id,computed,nullzero"`
	LastKnownLocationName      string                      `json:"lastKnownLocationName"      bun:"last_known_location_name,type:VARCHAR(255),scanonly"                                  keel:"last_known_location_name,computed,nullzero"`
	LastKnownLocationLatitude  *float64                    `json:"lastKnownLocationLatitude"  bun:"last_known_location_latitude,type:FLOAT,scanonly"                                     keel:"last_known_location_latitude,computed"`
	LastKnownLocationLongitude *float64                    `json:"lastKnownLocationLongitude" bun:"last_known_location_longitude,type:FLOAT,scanonly"                                    keel:"last_known_location_longitude,computed"`
	Version                    int64                       `json:"version"                    bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt                  int64                       `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                  int64                       `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	CustomFields map[string]any `json:"customFields,omitempty" bun:"-"`

	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          bun:"rel:belongs-to,join:business_unit_id=id"          keel:"belongs:business_unit_id"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          bun:"rel:belongs-to,join:organization_id=id"           keel:"belongs:organization_id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         bun:"rel:belongs-to,join:equipment_type_id=id"         keel:"belongs:equipment_type_id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id" keel:"belongs:equipment_manufacturer_id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             bun:"rel:belongs-to,join:fleet_code_id=id"             keel:"belongs:fleet_code_id"`
	State                 *usstate.UsState                             `json:"state,omitempty"                 bun:"rel:belongs-to,join:state_id=id"                  keel:"belongs:state_id"`
	PrimaryWorker         *worker.Worker                               `json:"primaryWorker,omitempty"         bun:"rel:belongs-to,join:primary_worker_id=id"         keel:"belongs:primary_worker_id"`
	SecondaryWorker       *worker.Worker                               `json:"secondaryWorker,omitempty"       bun:"rel:belongs-to,join:secondary_worker_id=id"       keel:"belongs:secondary_worker_id"`
	OwnerWorker           *worker.Worker                               `json:"ownerWorker,omitempty"           bun:"rel:belongs-to,join:owner_worker_id=id"           keel:"belongs:owner_worker_id"`
}

func (t *Tractor) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
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
		validation.Field(
			&t.FuelType,
			domainvalidation.ValidEnum[domaintypes.IFTAFuelType]("Fuel type is invalid"),
		),
	))

	if t.OwnershipType != "" && !t.OwnershipType.IsValid() {
		multiErr.Add("ownershipType", errortypes.ErrInvalid, "Ownership type is invalid")
	}
	if t.OwnershipType == domaintypes.OwnershipTypeOwnerOperator &&
		(t.OwnerWorkerID == nil || t.OwnerWorkerID.IsNil()) {
		multiErr.Add(
			"ownerWorkerId",
			errortypes.ErrRequired,
			"Owner is required for owner-operator equipment",
		)
	}
	if t.OwnershipType != domaintypes.OwnershipTypeOwnerOperator && t.OwnerWorkerID != nil &&
		!t.OwnerWorkerID.IsNil() {
		multiErr.Add(
			"ownerWorkerId",
			errortypes.ErrInvalid,
			"Owner may only be set on owner-operator equipment",
		)
	}
	if t.OwnershipType == domaintypes.OwnershipTypeLeased && t.LessorName == "" {
		multiErr.Add(
			"lessorName",
			errortypes.ErrRequired,
			"Lessor name is required for leased equipment",
		)
	}
}

func (t *Tractor) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if t.FuelType == "" {
		t.FuelType = domaintypes.IFTAFuelTypeDiesel
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if t.OwnershipType == "" {
			t.OwnershipType = domaintypes.OwnershipTypeCompanyOwned
		}
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

func (t *Tractor) GetCreatedAt() int64 {
	return t.CreatedAt
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
