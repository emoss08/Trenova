package locationcategory

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*LocationCategory)(nil)
	_ validationframework.TenantedEntity = (*LocationCategory)(nil)
	_ domaintypes.PostgresSearchable     = (*LocationCategory)(nil)
)

type LocationCategory struct {
	bun.BaseModel `bun:"table:location_categories,alias:lc" json:"-"`

	ID                  pulid.ID     `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID     `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID      pulid.ID     `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	Name                string       `json:"name"                bun:"name,type:VARCHAR(100),notnull"`
	Description         string       `json:"description"         bun:"description,type:TEXT,nullzero"`
	Type                Category     `json:"type"                bun:"type,type:location_category_type,notnull"`
	FacilityType        FacilityType `json:"facilityType"        bun:"facility_type,type:facility_type,nullzero"`
	Color               string       `json:"color"               bun:"color,type:VARCHAR(10),nullzero"`
	HasSecureParking    bool         `json:"hasSecureParking"    bun:"has_secure_parking,type:BOOLEAN,default:false"`
	RequiresAppointment bool         `json:"requiresAppointment" bun:"requires_appointment,type:BOOLEAN,default:false"`
	AllowsOvernight     bool         `json:"allowsOvernight"     bun:"allows_overnight,type:BOOLEAN,default:false"`
	HasRestroom         bool         `json:"hasRestroom"         bun:"has_restroom,type:BOOLEAN,default:false"`
	Version             int64        `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64        `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64        `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector        string       `json:"-"                   bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                string       `json:"-"                   bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (lc *LocationCategory) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(lc,
		validation.Field(&lc.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&lc.Type,
			validation.Required.Error("Type is required"),
			validation.By(func(value any) error {
				if v, ok := value.(Category); ok && !v.IsValid() {
					return errors.New("invalid type")
				}
				return nil
			}),
		),
		validation.Field(&lc.FacilityType,
			validation.By(func(value any) error {
				if v, ok := value.(FacilityType); ok && v != "" && !v.IsValid() {
					return errors.New("invalid facility type")
				}
				return nil
			}),
		),
		validation.Field(&lc.Color,
			is.HexColor.Error("Color must be a valid hex color"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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

func (lc *LocationCategory) GetID() pulid.ID {
	return lc.ID
}

func (lc *LocationCategory) GetOrganizationID() pulid.ID {
	return lc.OrganizationID
}

func (lc *LocationCategory) GetBusinessUnitID() pulid.ID {
	return lc.BusinessUnitID
}

func (lc *LocationCategory) GetTableName() string {
	return "location_categories"
}

func (lc *LocationCategory) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "lc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}
