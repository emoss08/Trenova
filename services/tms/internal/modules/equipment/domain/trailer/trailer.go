package trailer

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
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable     = (*Trailer)(nil)
	_ customfield.CustomFieldsSupporter  = (*Trailer)(nil)
	_ pagination.CursorEntity            = (*Trailer)(nil)
	_ validationframework.TenantedEntity = (*Trailer)(nil)
)

// Trailers and towed equipment
type Trailer struct {
	keel.Table `keel:"trailers,report=Equipment"`

	ID                         pulid.ID                    `json:"id"                         keel:"id,pk,autoid:tr_"`
	BusinessUnitID             pulid.ID                    `json:"businessUnitId"             keel:"business_unit_id,pk,tenant"`
	OrganizationID             pulid.ID                    `json:"organizationId"             keel:"organization_id,pk,tenant"`
	EquipmentTypeID            pulid.ID                    `json:"equipmentTypeId"            keel:"equipment_type_id"`
	EquipmentManufacturerID    pulid.ID                    `json:"equipmentManufacturerId"    keel:"equipment_manufacturer_id"`
	RegistrationStateID        pulid.ID                    `json:"registrationStateId"        keel:"registration_state_id,nullzero"`
	FleetCodeID                pulid.ID                    `json:"fleetCodeId"                keel:"fleet_code_id,nullzero"`
	Status                     domaintypes.EquipmentStatus `json:"status"                     keel:"status"`
	Code                       string                      `json:"code"                       keel:"code,label=Unit Number"`
	Model                      string                      `json:"model"                      keel:"model,nullzero"`
	Make                       string                      `json:"make"                       keel:"make,nullzero"`
	Year                       *int                        `json:"year"                       keel:"year,dimension"`
	LicensePlateNumber         string                      `json:"licensePlateNumber"         keel:"license_plate_number,nullzero,label=License Plate"`
	Vin                        string                      `json:"vin"                        keel:"vin,nullzero"`
	ExternalID                 string                      `json:"externalId"                 keel:"external_id,nullzero"`
	RegistrationNumber         string                      `json:"registrationNumber"         keel:"registration_number,nullzero"`
	MaxLoadWeight              *int                        `json:"maxLoadWeight"              keel:"max_load_weight,format=weight"`
	OwnershipType              domaintypes.OwnershipType   `json:"ownershipType"              keel:"ownership_type"`
	OwnerWorkerID              *pulid.ID                   `json:"ownerWorkerId"              keel:"owner_worker_id"`
	LessorName                 string                      `json:"lessorName"                 keel:"lessor_name,nullzero"`
	LeaseReference             string                      `json:"leaseReference"             keel:"lease_reference,nullzero"`
	LeaseEndDate               *int64                      `json:"leaseEndDate"               keel:"lease_end_date"`
	LastInspectionDate         *int64                      `json:"lastInspectionDate"         keel:"last_inspection_date,label=Last Inspection"`
	RegistrationExpiry         *int64                      `json:"registrationExpiry"         keel:"registration_expiry,type=epoch"`
	LastKnownLocationID        pulid.ID                    `json:"lastKnownLocationId"        keel:"last_known_location_id,computed,nullzero"`
	LastKnownLocationName      string                      `json:"lastKnownLocationName"      keel:"last_known_location_name,computed,nullzero"`
	LastKnownLocationLatitude  *float64                    `json:"lastKnownLocationLatitude"  keel:"last_known_location_latitude,computed"`
	LastKnownLocationLongitude *float64                    `json:"lastKnownLocationLongitude" keel:"last_known_location_longitude,computed"`
	SearchVector               string                      `json:"-"                          keel:"search_vector,tsvector,readonly,noselect"`
	Rank                       string                      `json:"-"                          keel:"rank,computed"`
	Version                    int64                       `json:"version"                    keel:"version,version"`
	CreatedAt                  int64                       `json:"createdAt"                  keel:"created_at,autocreate"`
	UpdatedAt                  int64                       `json:"updatedAt"                  keel:"updated_at,autoupdate"`
	CustomFields               map[string]any              `json:"customFields,omitempty"`

	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          keel:"belongs:business_unit_id,noreport"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          keel:"belongs:organization_id,noreport"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         keel:"belongs:equipment_type_id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" keel:"belongs:equipment_manufacturer_id,label=Manufacturer"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             keel:"belongs:fleet_code_id"`
	RegistrationState     *usstate.UsState                             `json:"registrationState,omitempty"     keel:"belongs:registration_state_id,noreport"`
	OwnerWorker           *worker.Worker                               `json:"ownerWorker,omitempty"           keel:"belongs:owner_worker_id"`
}

func (t *Trailer) BeforeCreate(_ context.Context) error {
	if t.OwnershipType == "" {
		t.OwnershipType = domaintypes.OwnershipTypeCompanyOwned
	}
	if t.Status == "" {
		t.Status = domaintypes.EquipmentStatusAvailable
	}

	return nil
}

func (t *Trailer) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		t,
		validation.Field(&t.Code, validation.Required),
		validation.Field(
			&t.Code,
			validation.Length(1, 50).Error("Code must be between 1 and 50 characters"),
		),
		validation.Field(
			&t.Make,
			validation.Length(1, 50).Error("Make must be between 1 and 50 characters"),
		),
		validation.Field(
			&t.Model,
			validation.Length(1, 50).Error("Model must be between 1 and 50 characters"),
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
			&t.Vin,
			validation.By(domaintypes.ValidateVin),
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

func (t *Trailer) GetID() pulid.ID {
	return t.ID
}

func (t *Trailer) GetCreatedAt() int64 {
	return t.CreatedAt
}

func (t *Trailer) GetOrganizationID() pulid.ID {
	return t.OrganizationID
}

func (t *Trailer) GetBusinessUnitID() pulid.ID {
	return t.BusinessUnitID
}

func (t *Trailer) GetTableName() string {
	return "trailers"
}

func (t *Trailer) GetResourceType() string {
	return "trailer"
}

func (t *Trailer) GetResourceID() string {
	return t.ID.String()
}

func (t *Trailer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "tr",
		ComputedFilters: []string{"lastKnownLocationId"},
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
		Relationships: []*domaintypes.RelationshipDefintion{
			{Field: "EquipmentType", Queryable: true},
			{Field: "EquipmentManufacturer", Queryable: true},
			{Field: "FleetCode", Queryable: true},
		},
	}
}
