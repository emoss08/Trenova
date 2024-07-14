package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type DocumentClassificationPermission string

const (
	// PermissionDocumentClassificationView is the permission to view document classification details
	PermissionDocumentClassificationView = DocumentClassificationPermission("documentclassification.view")

	// PermissionDocumentClassificationEdit is the permission to edit document classification details
	PermissionDocumentClassificationEdit = DocumentClassificationPermission("documentclassification.edit")

	// PermissionDocumentClassificationAdd is the permission to add a necw document classification
	PermissionDocumentClassificationAdd = DocumentClassificationPermission("documentclassification.add")

	// PermissionDocumentClassificationDelete is the permission to delete an document classification
	PermissionDocumentClassificationDelete = DocumentClassificationPermission("documentclassification.delete")
)

// String returns the string representation of the DocumentClassificationPermission
func (p DocumentClassificationPermission) String() string {
	return string(p)
}

type DocumentClassification struct {
	bun.BaseModel `bun:"table:document_classifications,alias:dc" json:"-"`

	ID          uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status      property.Status `bun:"status,type:status" json:"status"`
	Code        string          `bun:"type:VARCHAR(10),notnull" json:"code" queryField:"true"`
	Description string          `bun:"type:TEXT" json:"description"`
	Color       string          `bun:"type:VARCHAR(10)" json:"color"`
	Version     int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt   time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
}

func (c DocumentClassification) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Code, validation.Required, validation.Length(1, 10).Error("Code must be atleast 10 characters")),
		validation.Field(&c.Color, is.HexColor),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

func (c *DocumentClassification) BeforeUpdate(_ context.Context) error {
	c.Version++

	return nil
}

func (c *DocumentClassification) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
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
			Message: fmt.Sprintf("Version mismatch. The DocumentClassification (ID: %s) has been updated by another user. Please refresh and try again.", c.ID),
		}
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*DocumentClassification)(nil)

func (c *DocumentClassification) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
