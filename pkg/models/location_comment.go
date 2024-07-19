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

	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type LocationComment struct {
	bun.BaseModel `bun:"table:location_comments,alias:lc" json:"-"`

	ID        uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Comment   string    `bun:"type:TEXT,notnull" json:"comment"`
	Version   int64     `bun:"type:BIGINT" json:"version"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	CommentTypeID  uuid.UUID `bun:"type:uuid,notnull" json:"commentTypeId"`
	LocationID     uuid.UUID `bun:"type:uuid,notnull" json:"locationId"`
	UserID         uuid.UUID `bun:"type:uuid,notnull" json:"userId"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	CommentType  *CommentType  `bun:"rel:belongs-to,join:comment_type_id=id" json:"-"`
	User         *User         `bun:"rel:belongs-to,join:user_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (l LocationComment) Validate() error {
	return validation.ValidateStruct(
		&l,
		validation.Field(&l.BusinessUnitID, validation.Required),
		validation.Field(&l.OrganizationID, validation.Required),
		validation.Field(&l.LocationID, validation.Required),
		validation.Field(&l.UserID, validation.Required),
		validation.Field(&l.CommentTypeID, validation.Required),
	)
}

func (l *LocationComment) BeforeUpdate(_ context.Context) error {
	l.Version++

	return nil
}

func (l *LocationComment) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The LocationComment (ID: %s) has been updated by another user. Please refresh and try again.", l.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Location)(nil)

func (l *LocationComment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		l.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		l.UpdatedAt = time.Now()
	}
	return nil
}
