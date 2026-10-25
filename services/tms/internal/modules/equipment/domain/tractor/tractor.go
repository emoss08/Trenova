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
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable     = (*Tractor)(nil)
	_ customfield.CustomFieldsSupporter  = (*Tractor)(nil)
	_ pagination.CursorEntity            = (*Tractor)(nil)
	_ validationframework.TenantedEntity = (*Tractor)(nil)
)

// Power units
type Tractor struct {
	keel.Table `keel:"tractors,report=Equipment"`

	ID                         pulid.ID                    `json:"id"                         keel:"id,pk,autoid:trac_"`
	BusinessUnitID             pulid.ID                    `json:"businessUnitId"             keel:"business_unit_id,pk,tenant"`
	OrganizationID             pulid.ID                    `json:"organizationId"             keel:"organization_id,pk,tenant"`
	PrimaryWorkerID            pulid.ID                    `json:"primaryWorkerId"            keel:"primary_worker_id"`
	EquipmentTypeID            pulid.ID                    `json:"equipmentTypeId"            keel:"equipment_type_id"`
	EquipmentManufacturerID    pulid.ID                    `json:"equipmentManufacturerId"    keel:"equipment_manufacturer_id"`
	StateID                    pulid.ID                    `json:"stateId"                    keel:"state_id,nullzero"`
	FleetCodeID                pulid.ID                    `json:"fleetCodeId"                keel:"fleet_code_id,nullzero"`
	SecondaryWorkerID          pulid.ID                    `json:"secondaryWorkerId"          keel:"secondary_worker_id,nullzero"`
	Status                     domaintypes.EquipmentStatus `json:"status"                     keel:"status"`
	Code                       string                      `json:"code"                       keel:"code,label=Unit Number"`
	Model                      string                      `json:"model"                      keel:"model,nullzero"`
	Make                       string                      `json:"make"                       keel:"make,nullzero"`
	Year                       *int                        `json:"year"                       keel:"year,dimension"`
	LicensePlateNumber         string                      `json:"licensePlateNumber"         keel:"license_plate_number,nullzero,label=License Plate"`
	RegistrationNumber         string                      `json:"registrationNumber"         keel:"registration_number,nullzero"`
	RegistrationExpiry         *int64                      `json:"registrationExpiry"         keel:"registration_expiry,type=epoch"`
	FuelType                   domaintypes.IFTAFuelType    `json:"fuelType"                   keel:"fuel_type"`
	IFTAQualified              bool                        `json:"iftaQualified"              keel:"ifta_qualified"`
	Vin                        string                      `json:"vin"                        keel:"vin,nullzero"`
	ExternalID                 string                      `json:"externalId"                 keel:"external_id,nullzero"`
	OwnershipType              domaintypes.OwnershipType   `json:"ownershipType"              keel:"ownership_type"`
	OwnerWorkerID              *pulid.ID                   `json:"ownerWorkerId"              keel:"owner_worker_id"`
	LessorName                 string                      `json:"lessorName"                 keel:"lessor_name,nullzero"`
	LeaseReference             string                      `json:"leaseReference"             keel:"lease_reference,nullzero"`
	LeaseEndDate               *int64                      `json:"leaseEndDate"               keel:"lease_end_date"`
	LastKnownLocationID        pulid.ID                    `json:"lastKnownLocationId"        keel:"last_known_location_id,computed,nullzero"`
	LastKnownLocationName      string                      `json:"lastKnownLocationName"      keel:"last_known_location_name,computed,nullzero"`
	LastKnownLocationLatitude  *float64                    `json:"lastKnownLocationLatitude"  keel:"last_known_location_latitude,computed"`
	LastKnownLocationLongitude *float64                    `json:"lastKnownLocationLongitude" keel:"last_known_location_longitude,computed"`
	Version                    int64                       `json:"version"                    keel:"version,version"`
	CreatedAt                  int64                       `json:"createdAt"                  keel:"created_at,autocreate"`
	UpdatedAt                  int64                       `json:"updatedAt"                  keel:"updated_at,autoupdate"`

	CustomFields map[string]any `json:"customFields,omitempty"`

	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          keel:"belongs:business_unit_id,noreport"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          keel:"belongs:organization_id,noreport"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         keel:"belongs:equipment_type_id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" keel:"belongs:equipment_manufacturer_id,label=Manufacturer"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             keel:"belongs:fleet_code_id"`
	State                 *usstate.UsState                             `json:"state,omitempty"                 keel:"belongs:state_id,noreport"`
	PrimaryWorker         *worker.Worker                               `json:"primaryWorker,omitempty"         keel:"belongs:primary_worker_id"`
	SecondaryWorker       *worker.Worker                               `json:"secondaryWorker,omitempty"       keel:"belongs:secondary_worker_id"`
	OwnerWorker           *worker.Worker                               `json:"ownerWorker,omitempty"           keel:"belongs:owner_worker_id"`
}

func (t *Tractor) BeforeUpdate(_ context.Context) error {
	if t.FuelType == "" {
		t.FuelType = domaintypes.IFTAFuelTypeDiesel
	}

	return nil
}

func (t *Tractor) BeforeCreate(_ context.Context) error {
	if t.FuelType == "" {
		t.FuelType = domaintypes.IFTAFuelTypeDiesel
	}
	if t.OwnershipType == "" {
		t.OwnershipType = domaintypes.OwnershipTypeCompanyOwned
	}
	if t.Status == "" {
		t.Status = domaintypes.EquipmentStatusAvailable
	}

	return nil
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
		ComputedFilters: []string{"lastKnownLocationId"},
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
