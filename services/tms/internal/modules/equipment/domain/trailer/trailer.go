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
	"github.com/emoss08/trenova/pkg/dbtype"
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
	_ bun.BeforeAppendModelHook          = (*Trailer)(nil)
	_ domaintypes.PostgresSearchable     = (*Trailer)(nil)
	_ customfield.CustomFieldsSupporter  = (*Trailer)(nil)
	_ pagination.CursorEntity            = (*Trailer)(nil)
	_ validationframework.TenantedEntity = (*Trailer)(nil)
)

type Trailer struct {
	bun.BaseModel `bun:"table:trailers,alias:tr" json:"-"`
	keel.Table    `                                       keel:"trailers"`

	ID                         pulid.ID                    `json:"id"                         bun:"id,type:VARCHAR(100),pk,notnull"                                                      keel:"id,pk,autoid:tr_"`
	BusinessUnitID             pulid.ID                    `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                                        keel:"business_unit_id,pk,tenant"`
	OrganizationID             pulid.ID                    `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),notnull,pk"                                         keel:"organization_id,pk,tenant"`
	EquipmentTypeID            pulid.ID                    `json:"equipmentTypeId"            bun:"equipment_type_id,type:VARCHAR(100),notnull"                                          keel:"equipment_type_id"`
	EquipmentManufacturerID    pulid.ID                    `json:"equipmentManufacturerId"    bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull"                                  keel:"equipment_manufacturer_id"`
	RegistrationStateID        pulid.ID                    `json:"registrationStateId"        bun:"registration_state_id,type:VARCHAR(100),nullzero"                                     keel:"registration_state_id,nullzero"`
	FleetCodeID                pulid.ID                    `json:"fleetCodeId"                bun:"fleet_code_id,type:VARCHAR(100),nullzero"                                             keel:"fleet_code_id,nullzero"`
	Status                     domaintypes.EquipmentStatus `json:"status"                     bun:"status,type:equipment_status_enum,notnull,default:'Available'"                        keel:"status"`
	Code                       string                      `json:"code"                       bun:"code,type:VARCHAR(50),notnull"                                                        keel:"code"`
	Model                      string                      `json:"model"                      bun:"model,type:VARCHAR(50),nullzero"                                                      keel:"model,nullzero"`
	Make                       string                      `json:"make"                       bun:"make,type:VARCHAR(50),nullzero"                                                       keel:"make,nullzero"`
	Year                       *int                        `json:"year"                       bun:"year,type:INT,nullzero"                                                               keel:"year"`
	LicensePlateNumber         string                      `json:"licensePlateNumber"         bun:"license_plate_number,type:VARCHAR(50),nullzero"                                       keel:"license_plate_number,nullzero"`
	Vin                        string                      `json:"vin"                        bun:"vin,type:vin_code_optional,nullzero"                                                  keel:"vin,nullzero"`
	ExternalID                 string                      `json:"externalId"                 bun:"external_id,type:TEXT,nullzero"                                                       keel:"external_id,nullzero"`
	RegistrationNumber         string                      `json:"registrationNumber"         bun:"registration_number,type:VARCHAR(50),nullzero"                                        keel:"registration_number,nullzero"`
	MaxLoadWeight              *int                        `json:"maxLoadWeight"              bun:"max_load_weight,type:INT,nullzero"                                                    keel:"max_load_weight"`
	OwnershipType              domaintypes.OwnershipType   `json:"ownershipType"              bun:"ownership_type,type:VARCHAR(50),notnull,default:'CompanyOwned'"                       keel:"ownership_type"`
	OwnerWorkerID              *pulid.ID                   `json:"ownerWorkerId"              bun:"owner_worker_id,type:VARCHAR(100),nullzero"                                           keel:"owner_worker_id"`
	LessorName                 string                      `json:"lessorName"                 bun:"lessor_name,type:VARCHAR(150),nullzero"                                               keel:"lessor_name,nullzero"`
	LeaseReference             string                      `json:"leaseReference"             bun:"lease_reference,type:VARCHAR(100),nullzero"                                           keel:"lease_reference,nullzero"`
	LeaseEndDate               *int64                      `json:"leaseEndDate"               bun:"lease_end_date,type:BIGINT,nullzero"                                                  keel:"lease_end_date"`
	LastInspectionDate         *int64                      `json:"lastInspectionDate"         bun:"last_inspection_date,type:BIGINT,nullzero"                                            keel:"last_inspection_date"`
	RegistrationExpiry         *int64                      `json:"registrationExpiry"         bun:"registration_expiry,type:BIGINT,nullzero"                                             keel:"registration_expiry"`
	LastKnownLocationID        pulid.ID                    `json:"lastKnownLocationId"        bun:"last_known_location_id,type:VARCHAR(100),scanonly"                                    keel:"last_known_location_id,computed,nullzero"`
	LastKnownLocationName      string                      `json:"lastKnownLocationName"      bun:"last_known_location_name,type:VARCHAR(255),scanonly"                                  keel:"last_known_location_name,computed,nullzero"`
	LastKnownLocationLatitude  *float64                    `json:"lastKnownLocationLatitude"  bun:"last_known_location_latitude,type:FLOAT,scanonly"                                     keel:"last_known_location_latitude,computed"`
	LastKnownLocationLongitude *float64                    `json:"lastKnownLocationLongitude" bun:"last_known_location_longitude,type:FLOAT,scanonly"                                    keel:"last_known_location_longitude,computed"`
	SearchVector               string                      `json:"-"                          bun:"search_vector,type:TSVECTOR,scanonly"                                                 keel:"search_vector,tsvector,readonly,noselect"`
	Rank                       string                      `json:"-"                          bun:"rank,type:VARCHAR(100),scanonly"                                                      keel:"rank,computed"`
	Version                    int64                       `json:"version"                    bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt                  int64                       `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                  int64                       `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	CustomFields               map[string]any              `json:"customFields,omitempty"     bun:"-"`

	BusinessUnit          *tenant.BusinessUnit                         `json:"businessUnit,omitempty"          bun:"rel:belongs-to,join:business_unit_id=id"          keel:"belongs:business_unit_id"`
	Organization          *tenant.Organization                         `json:"organization,omitempty"          bun:"rel:belongs-to,join:organization_id=id"           keel:"belongs:organization_id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty"         bun:"rel:belongs-to,join:equipment_type_id=id"         keel:"belongs:equipment_type_id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id" keel:"belongs:equipment_manufacturer_id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty"             bun:"rel:belongs-to,join:fleet_code_id=id"             keel:"belongs:fleet_code_id"`
	RegistrationState     *usstate.UsState                             `json:"registrationState,omitempty"     bun:"rel:belongs-to,join:registration_state_id=id"     keel:"belongs:registration_state_id"`
	OwnerWorker           *worker.Worker                               `json:"ownerWorker,omitempty"           bun:"rel:belongs-to,join:owner_worker_id=id"           keel:"belongs:owner_worker_id"`
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

func (t *Trailer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.OwnershipType == "" {
			t.OwnershipType = domaintypes.OwnershipTypeCompanyOwned
		}
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("tr_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
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
			{
				Field:        "EquipmentType",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*equipmenttype.EquipmentType)(nil),
				TargetTable:  "equipment_types",
				ForeignKey:   "equipment_type_id",
				ReferenceKey: "id",
				Alias:        "et",
				Queryable:    true,
			},
			{
				Field:        "EquipmentManufacturer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*equipmentmanufacturer.EquipmentManufacturer)(nil),
				TargetTable:  "equipment_manufacturers",
				ForeignKey:   "equipment_manufacturer_id",
				ReferenceKey: "id",
				Alias:        "em",
				Queryable:    true,
			},
			{
				Field:        "FleetCode",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*fleetcode.FleetCode)(nil),
				TargetTable:  "fleet_codes",
				ForeignKey:   "fleet_code_id",
				ReferenceKey: "id",
				Alias:        "fc",
				Queryable:    true,
			},
		},
	}
}
