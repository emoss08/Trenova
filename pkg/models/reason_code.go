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

type ReasonCodePermission string

const (
	// PermissionReasonCodeView is the permission to view reason code details
	PermissionReasonCodeView = ReasonCodePermission("reasoncode.view")

	// PermissionReasonCodeEdit is the permission to edit reason code details
	PermissionReasonCodeEdit = ReasonCodePermission("reasoncode.edit")

	// PermissionReasonCodeAdd is the permission to add a necw reason code
	PermissionReasonCodeAdd = ReasonCodePermission("reasoncode.add")

	// PermissionReasonCodeDelete is the permission to delete an reason code
	PermissionReasonCodeDelete = ReasonCodePermission("reasoncode.delete")
)

// String returns the string representation of the ReasonCodePermission
func (p ReasonCodePermission) String() string {
	return string(p)
}

type ReasonCode struct {
	bun.BaseModel `bun:"table:reason_codes,alias:rc" json:"-"`

	ID          uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status      property.Status `bun:"status,type:status" json:"status"`
	Code        string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	CodeType    string          `bun:"type:VARCHAR(10)" json:"codeType"`
	Description string          `bun:"type:TEXT" json:"description"`
	Version     int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c ReasonCode) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(4, 4).Error("Code must be 4 characters")),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

func (c *ReasonCode) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *ReasonCode) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The ReasonCode (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*ReasonCode)(nil)

func (c *ReasonCode) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
