package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type BusinessUnit struct {
	bun.BaseModel `bun:"business_units"`
	CreatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name          string     `bun:"type:VARCHAR(100),notnull" json:"name" queryField:"true"`
	AddressLine1  string     `json:"addressLine1"`
	AddressLine2  string     `json:"addressLine2"`
	City          string     `json:"city"`
	StateID       *uuid.UUID `bun:"type:uuid" json:"stateId"`
	State         *UsState   `bun:"rel:belongs-to,join:state_id=id" json:"state"`
	PostalCode    string     `json:"postalCode"`
	PhoneNumber   string     `bun:"type:VARCHAR(15)" json:"phoneNumber"`
	ContactName   string     `json:"contactName"`
	ContactEmail  string     `json:"contactEmail"`
}

func (b BusinessUnit) Validate() error {
	return validation.ValidateStruct(
		&b,
		// Name is cannot be empty
		validation.Field(&b.Name, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*BusinessUnit)(nil)

func (b *BusinessUnit) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		b.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		b.UpdatedAt = time.Now()
	}
	return nil
}
