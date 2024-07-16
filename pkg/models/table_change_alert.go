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

type TableChangeAlert struct {
	bun.BaseModel `bun:"table:table_change_alerts,alias:tca" json:"-"`

	ID              uuid.UUID               `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status          property.Status         `bun:"status,type:status" json:"status"`
	Name            string                  `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	DatabaseAction  property.DatabaseAction `bun:"type:database_action_enum,notnull" json:"databaseAction"`
	TopicName       string                  `bun:"type:VARCHAR(200),notnull" json:"topicName"`
	Description     string                  `bun:"type:TEXT" json:"description"`
	CustomSubject   string                  `bun:"type:VARCHAR" json:"customSubject"`
	DeliveryMethod  property.DeliveryMethod `bun:"type:delivery_method_enum,notnull" json:"deliveryMethod"`
	EmailRecipients string                  `bun:"type:TEXT" json:"emailRecipients"`
	EffectiveDate   *pgtype.Date            `bun:"type:date" json:"effectiveDate"`
	ExpirationDate  *pgtype.Date            `bun:"type:date" json:"expirationDate"`
	Version         int64                   `bun:"type:BIGINT" json:"version"`
	CreatedAt       time.Time               `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt       time.Time               `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (f TableChangeAlert) Validate() error {
	return validation.ValidateStruct(
		&f,
		validation.Field(&f.Name, validation.Required),
		validation.Field(&f.DatabaseAction, validation.Required),
		validation.Field(&f.TopicName, validation.Required),
		validation.Field(&f.DeliveryMethod, validation.Required),
		validation.Field(&f.BusinessUnitID, validation.Required),
		validation.Field(&f.OrganizationID, validation.Required),
	)
}

func (f *TableChangeAlert) BeforeUpdate(_ context.Context) error {
	f.Version++

	return nil
}

func (f *TableChangeAlert) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := f.Version

	if err := f.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(f).
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
			Message: fmt.Sprintf("Version mismatch. The TableChangeAlert (ID: %s) has been updated by another user. Please refresh and try again.", f.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*TableChangeAlert)(nil)

func (f *TableChangeAlert) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		f.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		f.UpdatedAt = time.Now()
	}
	return nil
}
