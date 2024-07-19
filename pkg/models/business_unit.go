// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
