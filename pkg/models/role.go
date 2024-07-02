package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Role struct {
	bun.BaseModel  `bun:"roles"`
	CreatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name           string    `bun:",notnull" json:"name" queryField:"true"`
	Description    string    `bun:"type:TEXT" json:"description"`
	Color          string    `json:"color"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Permissions  []*Permission `bun:"m2m:role_permissions,join:Role=Permission" json:"permissions"`
}

func (r Role) Validate() error {
	return validation.ValidateStruct(
		&r,
		validation.Field(&r.Name, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*Role)(nil)

func (r *Role) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		r.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		r.UpdatedAt = time.Now()
	}
	return nil
}

type RolePermission struct {
	RoleID       uuid.UUID `bun:",pk,type:uuid" json:"roleId"`
	PermissionID uuid.UUID `bun:",pk,type:uuid" json:"permissionId"`

	Permission *Permission `bun:"rel:belongs-to,join:permission_id=id" json:"permission"`
	Role       *Role       `bun:"rel:belongs-to,join:role_id=id" json:"role"`
}
