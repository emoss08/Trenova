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
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Organization struct {
	bun.BaseModel `bun:"organizations" alias:"o" json:"-"`

	ID           uuid.UUID                 `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name         string                    `bun:"name,notnull" json:"name" queryField:"true"`
	ScacCode     string                    `bun:",notnull" json:"scacCode"`
	DOTNumber    string                    `bun:"dot_number" json:"dotNumber"`
	LogoURL      string                    `bun:"logo_url" json:"logoUrl"`
	OrgType      property.OrganizationType `bun:"org_type,notnull,default:'Asset'" json:"orgType"`
	AddressLine1 string                    `bun:"address_line_1" json:"addressLine1"`
	AddressLine2 string                    `bun:"address_line_2" json:"addressLine2"`
	City         string                    `bun:"city" json:"city"`
	PostalCode   string                    `bun:"postal_code" json:"postalCode"`
	Timezone     string                    `bun:"timezone,notnull,default:'America/New_York'" json:"timezone"`
	Version      int64                     `bun:"type:BIGINT" json:"version"`
	CreatedAt    time.Time                 `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt    time.Time                 `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid" json:"businessUnitId"`
	StateID        uuid.UUID `bun:"type:uuid,notnull" json:"stateId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	State        *UsState      `bun:"rel:belongs-to,join:state_id=id" json:"state"`
}

func (o Organization) Validate() error {
	return validation.ValidateStruct(
		&o,
		validation.Field(&o.BusinessUnitID, validation.Required),
		validation.Field(&o.Name, validation.Required),
		validation.Field(&o.ScacCode, validation.Required, validation.Length(4, 4).Error("SCAC code must be 4 characters")),
		// validation.Field(&o.DOTNumber, validation.Required, validation.Length(12, 12).Error("DOT number must be 12 characters")),
		validation.Field(&o.OrgType, validation.Required),
		validation.Field(&o.AddressLine1, validation.Length(0, 150).Error("Address line 1 must be less than 150 characters")),
		validation.Field(&o.City, validation.Required),
		validation.Field(&o.StateID, validation.Required),
		validation.Field(&o.Timezone, validation.Required),
	)
}

func (o *Organization) BeforeUpdate(_ context.Context) error {
	o.Version++

	return nil
}

func (o *Organization) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := o.Version

	if err := o.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(o).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The Organization (ID: %s) has been updated by another user. Please refresh and try again.", o.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Organization)(nil)

func (o *Organization) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		o.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		o.UpdatedAt = time.Now()
	}
	return nil
}
