package models

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationPermission string

const (
	// PermissionLocationView is the permission to view location details
	PermissionLocationView = LocationPermission("location.view")

	// PermissionLocationEdit is the permission to edit location details
	PermissionLocationEdit = LocationPermission("location.edit")

	// PermissionLocationAdd is the permission to add a necw location``
	PermissionLocationAdd = LocationPermission("location.add")

	// PermissionLocationDelete is the permission to delete an location``
	PermissionLocationDelete = LocationPermission("location.delete")
)

// String returns the string representation of the LocationPermission
func (p LocationPermission) String() string {
	return string(p)
}

type Location struct {
	bun.BaseModel      `bun:"table:locations,alias:lc" json:"-"`
	CreatedAt          time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt          time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID                 uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status             property.Status `bun:"status,type:status" json:"status"`
	Code               string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Name               string          `bun:"type:VARCHAR(255),notnull" json:"name"`
	AddressLine1       string          `bun:"address_line_1,type:VARCHAR(150),notnull" json:"addressLine1"`
	AddressLine2       string          `bun:"address_line_2,type:VARCHAR(150),notnull" json:"addressLine2"`
	City               string          `bun:"type:VARCHAR(150),notnull" json:"city"`
	PostalCode         string          `bun:"type:VARCHAR(10),notnull" json:"postalCode"`
	Longitude          float64         `bun:"type:float" json:"longitude"`
	Latitude           float64         `bun:"type:float" json:"latitude"`
	PlaceID            string          `bun:"type:VARCHAR(255)" json:"placeId"`
	IsGeocoded         bool            `bun:"type:boolean" json:"isGeocoded"`
	Description        string          `bun:"type:TEXT" json:"description"`
	LocationCategoryID uuid.UUID       `bun:"type:uuid,notnull" json:"locationCategoryId"`
	StateID            *uuid.UUID      `bun:"type:uuid,nullzero" json:"stateId"`
	BusinessUnitID     uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID     uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	LocationCategory *LocationCategory `bun:"rel:belongs-to,join:location_category_id=id" json:"locationCategory"`
	State            *UsState          `bun:"rel:belongs-to,join:state_id=id" json:"state"`
	BusinessUnit     *BusinessUnit     `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization     *Organization     `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (l Location) Validate() error {
	return validation.ValidateStruct(
		&l,
		validation.Field(&l.BusinessUnitID, validation.Required),
		validation.Field(&l.LocationCategoryID, validation.Required),
		validation.Field(&l.OrganizationID, validation.Required),
	)
}

func (l *Location) TableName() string {
	return "locations"
}

func (l *Location) GetCodePrefix(pattern string) string {
	switch pattern {
	case "NAME-COUNTER":
		return utils.TruncateString(strings.ToUpper(l.Name), 4)
	case "CITY-COUNTER":
		return utils.TruncateString(strings.ToUpper(l.City), 4)
	default:
		return utils.TruncateString(strings.ToUpper(l.Name), 4)
	}
}

func (l *Location) GenerateCode(pattern string, counter int) string {
	switch pattern {
	case "NAME-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.Name), 4), counter)
	case "CITY-COUNTER":
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.City), 4), counter)
	default:
		return fmt.Sprintf("%s%04d", utils.TruncateString(strings.ToUpper(l.Name), 4), counter)
	}
}

func (l *Location) InsertLocation(ctx context.Context, tx bun.Tx, codeGen *gen.CodeGenerator, pattern string) error {
	code, err := codeGen.GenerateUniqueCode(ctx, l, pattern, l.OrganizationID)
	if err != nil {
		return fmt.Errorf("error generating unique code: %w", err)
	}
	l.Code = code

	if err = l.Validate(); err != nil {
		return fmt.Errorf("location validation failed: %w", err)
	}

	_, err = tx.NewInsert().Model(l).Exec(ctx)
	if err != nil {
		return fmt.Errorf("error inserting location: %w", err)
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Location)(nil)

func (l *Location) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		l.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		l.UpdatedAt = time.Now()
	}
	return nil
}
