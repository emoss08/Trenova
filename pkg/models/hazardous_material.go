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

type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	ID                 uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name               string    `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	Status             string    `bun:"type:status_enum,notnull,default:'Active'" json:"status"`
	HazardClass        string    `bun:"type:VARCHAR(16),notnull:default:'HazardClass1And1'" json:"hazardClass"`
	ERGNumber          string    `bun:"type:VARCHAR" json:"ergNumber"`
	Description        string    `bun:"type:TEXT" json:"description"`
	PackingGroup       string    `bun:"type:VARCHAR,default:'PackingGroup1'" json:"packingGroup"`
	ProperShippingName string    `bun:"type:TEXT" json:"properShippingName"`
	Version            int64     `bun:"type:BIGINT" json:"version"`
	CreatedAt          time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt          time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (c HazardousMaterial) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

func (c *HazardousMaterial) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *HazardousMaterial) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := c.Version

	if err := c.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(c).
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
			Message: fmt.Sprintf("Version mismatch. The FleetCode (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*HazardousMaterial)(nil)

func (c *HazardousMaterial) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
