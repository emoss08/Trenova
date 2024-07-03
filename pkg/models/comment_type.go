package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CommentTypePermission string

const (
	// PermissionCommentTypeView is the permission to view comment type details
	PermissionCommentTypeView = CommentTypePermission("commenttype.view")

	// PermissionCommentTypeEdit is the permission to edit comment type details
	PermissionCommentTypeEdit = CommentTypePermission("commenttype.edit")

	// PermissionCommentTypeAdd is the permission to add a new comment type
	PermissionCommentTypeAdd = CommentTypePermission("commenttype.add")

	// PermissionCommentTypeDelete is the permission to delete an comment type
	PermissionCommentTypeDelete = CommentTypePermission("commenttype.delete")
)

// String returns the string representation of the CommentTypePermission
func (p CommentTypePermission) String() string {
	return string(p)
}

type CommentType struct {
	bun.BaseModel  `bun:"table:comment_types,alias:ct" json:"-"`
	CreatedAt      time.Time         `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time         `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID         `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name           string            `bun:"type:VARCHAR(20),notnull" json:"name" queryField:"true"`
	Description    string            `bun:"type:TEXT,notnull" json:"description"`
	Status         property.Status   `bun:"type:status_enum,notnull,default:'Active'" json:"status"`
	Severity       property.Severity `bun:"type:severity_enum,notnull,default:'Low'" json:"severity"`
	BusinessUnitID uuid.UUID         `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID         `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c CommentType) Validate() error {
	return validation.ValidateStruct(
		&c,
		validation.Field(&c.Name, validation.Required),
		validation.Field(&c.Description, validation.Required),
		validation.Field(&c.Severity, validation.Required),
		validation.Field(&c.BusinessUnitID, validation.Required),
		validation.Field(&c.OrganizationID, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*CommentType)(nil)

func (c *CommentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
