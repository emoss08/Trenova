package tractor

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Tractor)(nil)
	_ infra.SearchableEntity    = (*Tractor)(nil)
	_ domain.Validatable        = (*Tractor)(nil)
)

type Tractor struct {
	bun.BaseModel `bun:"table:tractors,alias:tr" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	EquipmentTypeID         pulid.ID  `bun:"equipment_type_id,type:VARCHAR(100),notnull" json:"equipmentTypeId"`
	PrimaryWorkerID         pulid.ID  `bun:"primary_worker_id,type:VARCHAR(100),notnull" json:"primaryWorkerId"`
	EquipmentManufacturerID pulid.ID  `bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull" json:"equipmentManufacturerId"`
	StateID                 *pulid.ID `bun:"state_id,type:VARCHAR(100),nullzero" json:"stateId"`
	FleetCodeID             *pulid.ID `bun:"fleet_code_id,type:VARCHAR(100),nullzero" json:"fleetCodeId"`
	SecondaryWorkerID       *pulid.ID `bun:"secondary_worker_id,type:VARCHAR(100),nullzero" json:"secondaryWorkerId"`

	// Core Fields
	Status             domain.EquipmentStatus `json:"status" bun:"status,type:equipment_status_enum,notnull,default:'Available'"`
	Code               string                 `json:"code" bun:"code,type:VARCHAR(50),notnull"`
	Model              string                 `json:"model" bun:"model,type:VARCHAR(50)"`
	Make               string                 `json:"make" bun:"make,type:VARCHAR(50)"`
	Year               *int                   `json:"year" bun:"year,type:INTEGER,nullzero"`
	LicensePlateNumber string                 `json:"licensePlateNumber" bun:"license_plate_number,type:VARCHAR(50)"`
	Vin                string                 `json:"vin" bun:"vin,type:vin_code_optional"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit          *businessunit.BusinessUnit                   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization          *organization.Organization                   `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	PrimaryWorker         *worker.Worker                               `json:"primaryWorker,omitempty" bun:"rel:belongs-to,join:primary_worker_id=id"`
	SecondaryWorker       *worker.Worker                               `json:"secondaryWorker,omitempty" bun:"rel:belongs-to,join:secondary_worker_id=id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty" bun:"rel:belongs-to,join:equipment_type_id=id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id"`
	State                 *usstate.UsState                             `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty" bun:"rel:belongs-to,join:fleet_code_id=id"`
}

func (t *Tractor) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, t,
		// Code is required and must be between 1 and 100 characters
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),

		// Equipment Type ID is required
		validation.Field(&t.EquipmentTypeID,
			validation.Required.Error("Equipment Type is required"),
		),

		// Primary Worker ID is required
		validation.Field(&t.PrimaryWorkerID,
			validation.Required.Error("Primary Worker is required"),
		),

		// Equipment Manufacturer ID is required
		validation.Field(&t.EquipmentManufacturerID,
			validation.Required.Error("Equipment Manufacturer is required"),
		),

		// Ensure VIN is valid.
		validation.Field(&t.Vin,
			validation.By(domain.ValidateVin),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (t *Tractor) GetID() string {
	return t.ID.String()
}

func (t *Tractor) GetTableName() string {
	return "tractors"
}

// Search Configuration
func (t *Tractor) GetSearchType() string {
	return "tractor"
}

func (t *Tractor) ToDocument() infra.SearchDocument {
	searchableText := []string{
		t.Code,
		t.Vin,
		t.LicensePlateNumber,
	}

	return infra.SearchDocument{
		ID:             t.ID.String(),
		Type:           "tractor",
		BusinessUnitID: t.BusinessUnitID.String(),
		OrganizationID: t.OrganizationID.String(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		Title:          t.Code,
		Description:    t.Code,
		SearchableText: strings.Join(searchableText, " "),
	}
}

func (t *Tractor) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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
