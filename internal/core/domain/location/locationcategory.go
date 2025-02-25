package location

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*LocationCategory)(nil)
	_ domain.Validatable        = (*LocationCategory)(nil)
)

type LocationCategory struct {
	bun.BaseModel `bun:"table:location_categories,alias:lc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Core Fields
	Name                string               `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Description         string               `json:"description" bun:"description,type:VARCHAR(255)"`
	Type                LocationCategoryType `json:"type" bun:"type,type:location_category_type,notnull"`
	FacilityType        FacilityType         `json:"facilityType" bun:"facility_type,type:facility_type,nullzero"`
	Color               string               `json:"color" bun:"color,type:VARCHAR(10)"`
	HasSecureParking    bool                 `json:"hasSecureParking" bun:"has_secure_parking,type:BOOLEAN,default:false"`
	RequiresAppointment bool                 `json:"requiresAppointment" bun:"requires_appointment,type:BOOLEAN,default:false"`
	AllowsOvernight     bool                 `json:"allowsOvernight" bun:"allows_overnight,type:BOOLEAN,default:false"`
	HasRestroom         bool                 `json:"hasRestroom" bun:"has_restroom,type:BOOLEAN,default:false"`

	// Metadata
	Version      int64  `bun:"version,type:BIGINT" json:"version"`
	CreatedAt    int64  `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt    int64  `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (lc *LocationCategory) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, lc,
		// Name is required and must be between 1 and 100 characters
		validation.Field(&lc.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		validation.Field(&lc.Type,
			validation.Required.Error("Type is required"),
			validation.In(
				LocationCategoryTypeTerminal,
				LocationCategoryTypeWarehouse,
				LocationCategoryTypeDistributionCenter,
				LocationCategoryTypeTruckStop,
				LocationCategoryTypeRestArea,
				LocationCategoryTypeCustomerLocation,
				LocationCategoryTypePort,
				LocationCategoryTypeRailYard,
				LocationCategoryTypeMaintenanceFacility,
			).Error("Invalid type"),
		),

		validation.Field(&lc.FacilityType,
			validation.In(
				FacilityTypeCrossDock,
				FacilityTypeStorageWarehouse,
				FacilityTypeColdStorage,
				FacilityTypeHazmatFacility,
				FacilityTypeIntermodalFacility,
			).Error("Invalid facility type"),
		),

		// Color must be a valid hex color
		validation.Field(&lc.Color,
			is.HexColor.Error("Color must be a valid hex color"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
		}
	}
}

// Pagination Configuration
func (lc *LocationCategory) GetID() string {
	return lc.ID.String()
}

func (lc *LocationCategory) GetTableName() string {
	return "location_categories"
}

func (lc *LocationCategory) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if lc.ID.IsNil() {
			lc.ID = pulid.MustNew("lc_")
		}

		lc.CreatedAt = now
	case *bun.UpdateQuery:
		lc.UpdatedAt = now
	}

	return nil
}
