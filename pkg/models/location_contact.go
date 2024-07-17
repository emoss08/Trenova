// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationContact struct {
	bun.BaseModel `bun:"table:location_contacts,alias:lca" json:"-"`

	ID           uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name         string    `bun:"type:VARCHAR(255),notnull" json:"name"`
	EmailAddress string    `bun:"type:VARCHAR(255)" json:"emailAddress"`
	PhoneNumber  string    `bun:"type:VARCHAR(20)" json:"phoneNumber"`
	Version      int64     `bun:"type:BIGINT" json:"version"`
	CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	LocationID     uuid.UUID `bun:"type:uuid,notnull" json:"locationId"`
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

func (l *LocationContact) BeforeUpdate(_ context.Context) error {
	l.Version++

	return nil
}

func (l *LocationContact) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := l.Version

	if err := l.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(l).
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
			Message: fmt.Sprintf("Version mismatch. The LocationContact (ID: %s) has been updated by another user. Please refresh and try again.", l.ID),
		}
	}

	return nil
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
