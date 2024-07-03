package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationContact struct {
	bun.BaseModel  `bun:"table:location_contacts,alias:lca" json:"-"`
	CreatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	LocationID     uuid.UUID `bun:"type:uuid,notnull" json:"locationId"`
	Name           string    `bun:"type:VARCHAR(255),notnull" json:"name"`
	EmailAddress   string    `bun:"type:VARCHAR(255)" json:"emailAddress"`
	PhoneNumber    string    `bun:"type:VARCHAR(20)" json:"phoneNumber"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (l LocationContact) Validate() error {
	return validation.ValidateStruct(
		&l,
		validation.Field(&l.BusinessUnitID, validation.Required),
		validation.Field(&l.OrganizationID, validation.Required),
		validation.Field(&l.LocationID, validation.Required),
		validation.Field(&l.Name, validation.Required),
		validation.Field(&l.PhoneNumber, validation.Length(1, 15)),
	)
}

var _ bun.BeforeAppendModelHook = (*Location)(nil)

func (l *LocationContact) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		l.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		l.UpdatedAt = time.Now()
	}
	return nil
}
