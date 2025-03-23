package location

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
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
	_ bun.BeforeAppendModelHook = (*Location)(nil)
	_ domain.Validatable        = (*Location)(nil)
	_ infra.PostgresSearchable  = (*Location)(nil)
)

type Location struct {
	bun.BaseModel `bun:"table:locations,alias:loc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// Relationship identifiers (Non-Primary-Keys)
	LocationCategoryID pulid.ID `json:"locationCategoryId" bun:"location_category_id,notnull,type:VARCHAR(100)"`
	StateID            pulid.ID `json:"stateId" bun:"state_id,notnull,type:VARCHAR(100)"`

	// Core Fields
	Status       domain.Status `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Code         string        `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Name         string        `json:"name" bun:"name,type:VARCHAR(255),notnull"`
	Description  string        `json:"description" bun:"description,type:VARCHAR(255)"`
	AddressLine1 string        `json:"addressLine1" bun:"address_line_1,type:VARCHAR(150),notnull"`
	AddressLine2 string        `json:"addressLine2" bun:"address_line_2,type:VARCHAR(150)"`
	City         string        `json:"city" bun:"city,type:VARCHAR(100),notnull"`
	PostalCode   string        `json:"postalCode" bun:"postal_code,type:us_postal_code,notnull"`
	Longitude    *float64      `json:"longitude" bun:"longitude,type:FLOAT,nullzero"`
	Latitude     *float64      `json:"latitude" bun:"latitude,type:FLOAT,nullzero"`
	PlaceID      string        `json:"placeId" bun:"place_id,type:VARCHAR(100)"`
	IsGeocoded   bool          `json:"isGeocoded" bun:"is_geocoded,type:BOOLEAN,default:false"`

	// Metadata
	Version      int64  `bun:"version,type:BIGINT" json:"version"`
	CreatedAt    int64  `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt    int64  `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit     *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization     *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	State            *usstate.UsState           `bun:"rel:belongs-to,join:state_id=id" json:"state,omitempty"`
	LocationCategory *LocationCategory          `bun:"rel:belongs-to,join:location_category_id=id" json:"locationCategory,omitempty"`
}

func (l *Location) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, l,
		validation.Field(&l.Code, validation.Required.Error("Code is required")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (l *Location) GetID() string {
	return l.ID.String()
}

func (l *Location) GetTableName() string {
	return "locations"
}

func (l *Location) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("loc_")
		}

		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}

func (l *Location) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "l",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "address_line_1",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "address_line_2",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "city",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "postal_code",
				Weight: "C",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "code || ' ' || name || ' ' || description || ' ' || address_line_1 || ' ' || address_line_2 || ' ' || city || ' ' || postal_code",
				Weight: "D",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
