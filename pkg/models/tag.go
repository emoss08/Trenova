package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TagPermission string

const (
	// TagView is the permission to view tag details
	PermissionTagView = TagPermission("tag.view")

	// TagEdit is the permission to edit tag details
	PermissionTagEdit = TagPermission("tag.edit")

	// TagAdd is the permission to add a new tag
	PermissionTagAdd = TagPermission("tag.add")

	// TagDelete is the permission to delete an tag
	PermissionTagDelete = TagPermission("tag.delete")
)

// String returns the string representation of the TagPermission
func (p TagPermission) String() string {
	return string(p)
}

type Tag struct {
	bun.BaseModel  `bun:"table:tags,alias:t" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status:default:'Active'" json:"status"`
	Name           string          `bun:"type:VARCHAR(50),notnull" json:"name" queryField:"true"`
	Description    string          `bun:"type:TEXT" json:"description"`
	Color          string          `bun:"type:VARCHAR(10)" json:"color"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

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
