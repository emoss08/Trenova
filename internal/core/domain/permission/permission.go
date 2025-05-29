package permission

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type Permission struct {
	bun.BaseModel `bun:"table:permissions,alias:perm"`

	ID               pulid.ID           `json:"id"                         bun:",pk,type:VARCHAR(100)"`
	Resource         Resource           `json:"resource"                   bun:"resource,type:VARCHAR(50),notnull"`
	Action           Action             `json:"action"                     bun:"action,type:action_enum,notnull"`
	Scope            Scope              `json:"scope"                      bun:"scope,type:scope_enum,notnull"`
	Description      string             `json:"description"                bun:"description,type:TEXT"`
	IsSystemLevel    bool               `json:"isSystemLevel"              bun:"is_system_level,notnull,default:false"`
	FieldPermissions []*FieldPermission `json:"fieldPermissions,omitempty" bun:"field_permissions,type:JSONB,default:'[]'::jsonb,nullzero"`
	Conditions       []*Condition       `json:"conditions,omitempty"       bun:"conditions,type:JSONB,default:'[]'::jsonb,nullzero"`
	Dependencies     []pulid.ID         `json:"dependencies"               bun:"dependencies,type:JSONB,default:'[]'::jsonb"`
	CustomSettings   map[string]any     `json:"customSettings,omitempty"   bun:"custom_settings,type:JSONB,default:'{}'::jsonb"`
	CreatedAt        int64              `json:"createdAt"                  bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64              `json:"updatedAt"                  bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (p *Permission) Validate() error {
	return validation.ValidateStruct(p,
		validation.Field(&p.Resource, validation.Required),
		validation.Field(&p.Action, validation.Required),
		validation.Field(&p.Scope, validation.Required),
		validation.Field(&p.Description, validation.Length(0, 1000)),
	)
}

var _ bun.BeforeAppendModelHook = (*Permission)(nil)

func (p *Permission) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if p.ID == "" {
			p.ID = pulid.MustNew("perm_")
		}
	}
	return nil
}
