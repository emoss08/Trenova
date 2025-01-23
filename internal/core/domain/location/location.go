package location

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/usstate"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Location)(nil)
	_ domain.Validatable        = (*Location)(nil)
)

type Location struct {
	bun.BaseModel `bun:"table:locations,alias:loc" json:"-"`

	// Primary identifiers
	ID                 pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID     pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID     pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`
	LocationCategoryID pulid.ID `bun:"location_category_id,notnull,type:VARCHAR(100)" json:"locationCategoryId"`
	StateID            pulid.ID `bun:"state_id,notnull,type:VARCHAR(100)" json:"stateId"`

	// Core Fields
	Status       domain.Status `json:"status" bun:"status,type:status_enum,notnull,default:'Active'"`
	Code         string        `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Name         string        `json:"name" bun:"name,type:VARCHAR(255),notnull"`
	Description  string        `json:"description" bun:"description,type:VARCHAR(255)"`
	AddressLine1 string        `json:"addressLine1" bun:"address_line_1,type:VARCHAR(150),notnull"`
	AddressLine2 string        `json:"addressLine2" bun:"address_line_2,type:VARCHAR(150)"`
	City         string        `json:"city" bun:"city,type:VARCHAR(100),notnull"`
	PostalCode   string        `json:"postalCode" bun:"postal_code,type:VARCHAR(10),notnull"`
	Longitude    *float64      `json:"longitude" bun:"longitude,type:FLOAT,nullzero"`
	Latitude     *float64      `json:"latitude" bun:"latitude,type:FLOAT,nullzero"`
	PlaceID      string        `json:"placeId" bun:"place_id,type:VARCHAR(100)"`
	IsGeocoded   bool          `json:"isGeocoded" bun:"is_geocoded,type:BOOLEAN,default:false"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

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
			errors.FromValidationErrors(validationErrs, multiErr, "")
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
