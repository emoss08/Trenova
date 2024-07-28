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

	"github.com/emoss08/trenova/pkg/audit"
	"github.com/emoss08/trenova/pkg/constants"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/uptrace/bun"
)

type WorkerProfile struct {
	bun.BaseModel `bun:"table:worker_profiles,alias:wkp" json:"-"`

	ID                    uuid.UUID                  `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	DateOfBirth           *pgtype.Date               `bun:"type:date,nullzero" json:"dateOfBirth"`
	LicenseNumber         string                     `bun:"type:VARCHAR(50),notnull" json:"licenseNumber"`
	Endorsements          property.WorkerEndorsement `bun:"type:worker_endorsement_enum,default:'None',notnull" json:"endorsements"`
	HazmatExpirationDate  *pgtype.Date               `bun:"type:date,nullzero" json:"hazmatExpirationDate"`
	LicenseExpirationDate *pgtype.Date               `bun:"type:date,nullzero" json:"licenseExpirationDate"`
	HireDate              *pgtype.Date               `bun:"type:date,nullzero" json:"hireDate"`
	TerminationDate       *pgtype.Date               `bun:"type:date,nullzero" json:"terminationDate"`
	PhysicalDueDate       *pgtype.Date               `bun:"type:date,nullzero" json:"physicalDueDate"`
	MVRDueDate            *pgtype.Date               `bun:"type:date,nullzero" json:"mvrDueDate"`
	Version               int64                      `bun:"type:BIGINT" json:"version"`
	CreatedAt             time.Time                  `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt             time.Time                  `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

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

func (wp *WorkerProfile) Insert(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	if err := wp.Validate(); err != nil {
		return err
	}

	if _, err := tx.NewInsert().Model(wp).Returning("*").Exec(ctx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableWorkerProfile,
		wp.ID.String(),
		property.AuditLogActionCreate,
		user,
		wp.OrganizationID,
		wp.BusinessUnitID,
		audit.WithDiff(nil, wp),
	)

	return nil
}

func (wp *WorkerProfile) UpdateOne(ctx context.Context, tx bun.IDB, auditService *audit.Service, user audit.AuditUser) error {
	original := new(WorkerProfile)
	if err := tx.NewSelect().Model(original).Where("id = ?", wp.ID).Scan(ctx); err != nil {
		return err
	}

	if err := wp.OptimisticUpdate(ctx, tx); err != nil {
		return err
	}

	auditService.LogAction(
		constants.TableWorkerProfile,
		wp.ID.String(),
		property.AuditLogActionUpdate,
		user,
		wp.OrganizationID,
		wp.BusinessUnitID,
		audit.WithDiff(original, wp),
	)

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
