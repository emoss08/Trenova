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

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wkp" json:"-"`

	ID                   uuid.UUID                  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	DateOfBirth          *pgtype.Date               `bun:"type:date,nullzero" json:"dateOfBirth"`
	LicenseNumber        string                     `bun:"type:VARCHAR(50),notnull" json:"licenseNumber"`
	Endorsements         property.WorkerEndorsement `bun:"type:worker_endorsement_enum,default:'None',notnull" json:"endorsements"`
	HazmatExpirationDate *pgtype.Date               `bun:"type:date,nullzero" json:"hazmatExpirationDate"`
	HireDate             *pgtype.Date               `bun:"type:date,nullzero" json:"hireDate"`
	TerminationDate      *pgtype.Date               `bun:"type:date,nullzero" json:"terminationDate"`
	PhysicalDueDate      *pgtype.Date               `bun:"type:date,nullzero" json:"physicalDueDate"`
	MVRDueDate           *pgtype.Date               `bun:"type:date,nullzero" json:"mvrDueDate"`
	Version              int64                      `bun:"type:BIGINT" json:"version"`
	CreatedAt            time.Time                  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt            time.Time                  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	StateID        *uuid.UUID `bun:"type:uuid,nullzero" json:"stateId"`
	WorkerID       uuid.UUID  `bun:"type:uuid,notnull" json:"workerId"`
	BusinessUnitID uuid.UUID  `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID  `bun:"type:uuid,notnull" json:"organizationId"`

	State        *UsState      `bun:"rel:belongs-to,join:state_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (wp WorkerProfile) Validate() error {
	return validation.ValidateStruct(
		&wp,
		validation.Field(&wp.WorkerID, validation.Required),
		validation.Field(&wp.BusinessUnitID, validation.Required),
		validation.Field(&wp.OrganizationID, validation.Required),
		validation.Field(&wp.Endorsements, validation.In(property.GetWorkerEndorsementList()...)),
	)
}

func (wp *WorkerProfile) BeforeUpdate(_ context.Context) error {
	wp.Version++

	return nil
}

func (wp *WorkerProfile) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := wp.Version

	if err := wp.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(wp).
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
			Message: fmt.Sprintf("Version mismatch. The Worker (ID: %s) has been updated by another user. Please refresh and try again.", wp.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*WorkerProfile)(nil)

func (wp *WorkerProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		wp.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		wp.UpdatedAt = time.Now()
	}
	return nil
}
