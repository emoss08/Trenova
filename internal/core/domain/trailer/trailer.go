package trailer

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
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Trailer)(nil)
	_ infra.SearchableEntity    = (*Trailer)(nil)
	_ domain.Validatable        = (*Trailer)(nil)
)

type Trailer struct {
	bun.BaseModel `bun:"table:trailers,alias:tr" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	EquipmentTypeID         pulid.ID  `bun:"equipment_type_id,type:VARCHAR(100),notnull" json:"equipmentTypeId"`
	EquipmentManufacturerID pulid.ID  `bun:"equipment_manufacturer_id,type:VARCHAR(100),notnull" json:"equipmentManufacturerId"`
	FleetCodeID             *pulid.ID `bun:"fleet_code_id,type:VARCHAR(100),nullzero" json:"fleetCodeId"`
	RegistrationStateID     *pulid.ID `bun:"registration_state_id,type:VARCHAR(100),nullzero" json:"registrationStateId"`

	// Core Fields
	Status             domain.EquipmentStatus `json:"status" bun:"status,type:equipment_status_enum,notnull,default:'Available'"`
	Code               string                 `json:"code" bun:"code,type:VARCHAR(50),notnull"`
	Model              string                 `json:"model" bun:"model,type:VARCHAR(50)"`
	Make               string                 `json:"make" bun:"make,type:VARCHAR(50)"`
	LicensePlateNumber string                 `json:"licensePlateNumber" bun:"license_plate_number,type:VARCHAR(50)"`
	Vin                string                 `json:"vin" bun:"vin,type:VARCHAR(50)"`
	RegistrationNumber string                 `json:"registrationNumber" bun:"registration_number,type:VARCHAR(50)"`
	Year               *int                   `json:"year" bun:"year,type:INTEGER,nullzero"`
	MaxLoadWeight      *int                   `json:"maxLoadWeight" bun:"max_load_weight,type:INTEGER,nullzero"`
	LastInspectionDate *int64                 `json:"lastInspectionDate" bun:"last_inspection_date,type:INTEGER,nullzero"`
	RegistrationExpiry *int64                 `json:"registrationExpiry" bun:"registration_expiry,type:INTEGER,nullzero"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit          *businessunit.BusinessUnit                   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization          *organization.Organization                   `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	EquipmentType         *equipmenttype.EquipmentType                 `json:"equipmentType,omitempty" bun:"rel:belongs-to,join:equipment_type_id=id"`
	EquipmentManufacturer *equipmentmanufacturer.EquipmentManufacturer `json:"equipmentManufacturer,omitempty" bun:"rel:belongs-to,join:equipment_manufacturer_id=id"`
	RegistrationState     *usstate.UsState                             `json:"state,omitempty" bun:"rel:belongs-to,join:registration_state_id=id"`
	FleetCode             *fleetcode.FleetCode                         `json:"fleetCode,omitempty" bun:"rel:belongs-to,join:fleet_code_id=id"`
}

func (t *Trailer) Validate(ctx context.Context, multiErr *errors.MultiError) {
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

		// Equipment Manufacturer ID is required
		validation.Field(&t.EquipmentManufacturerID,
			validation.Required.Error("Equipment Manufacturer is required"),
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
func (t *Trailer) GetID() string {
	return t.ID.String()
}

func (t *Trailer) GetTableName() string {
	return "trailers"
}

// Search Configuration
func (t *Trailer) GetSearchType() string {
	return "trailer"
}

func (t *Trailer) ToDocument() infra.SearchDocument {
	searchableText := []string{
		t.Code,
		t.Vin,
		t.LicensePlateNumber,
	}

	return infra.SearchDocument{
		ID:             t.ID.String(),
		Type:           "trailer",
		BusinessUnitID: t.BusinessUnitID.String(),
		OrganizationID: t.OrganizationID.String(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		Title:          t.Code,
		Description:    t.Code,
		SearchableText: strings.Join(searchableText, " "),
	}
}

func (t *Trailer) BeforeAppendModel(_ context.Context, query bun.Query) error {
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
