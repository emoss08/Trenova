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

type Tag struct {
	bun.BaseModel `bun:"table:tags,alias:t" json:"-"`
	ID            uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status        property.Status `bun:"status,type:status:default:'Active'" json:"status"`
	Name          string          `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	Description   string          `bun:"type:TEXT" json:"description"`
	Color         string          `bun:"type:VARCHAR(10)" json:"color"`
	Version       int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt     time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (t Tag) Validate() error {
	return validation.ValidateStruct(
		&t,
		validation.Field(&t.Name, validation.Required),
		validation.Field(&t.BusinessUnitID, validation.Required),
		validation.Field(&t.OrganizationID, validation.Required),
	)
}

func (t *Tag) BeforeUpdate(_ context.Context) error {
	t.Version++

	return nil
}

func (t *Tag) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := t.Version

	if err := t.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(t).
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
			Message: fmt.Sprintf("Version mismatch. The Tag (ID: %s) has been updated by another user. Please refresh and try again.", t.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*Tag)(nil)

func (t *Tag) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		t.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		t.UpdatedAt = time.Now()
	}
	return nil
}
