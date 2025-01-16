package permission

import (
	"context"

	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type Template struct {
	bun.BaseModel `bun:"table:permission_templates,alias:pt"`

	ID            pulid.ID          `json:"id" bun:",pk,type:VARCHAR(100)"`
	Name          string            `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Description   string            `json:"description" bun:"description,type:TEXT"`
	Permissions   []Permission      `json:"permissions" bun:"permissions,type:JSONB"`
	FieldSettings []FieldPermission `json:"fieldSettings" bun:"field_settings,type:JSONB"`
	IsSystem      bool              `json:"isSystem" bun:"is_system,notnull,default:false"`
	CreatedAt     int64             `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt     int64             `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

var _ bun.BeforeAppendModelHook = (*Template)(nil)

func (t *Template) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if t.ID == "" {
			t.ID = pulid.MustNew("pt_")
		}
	}
	return nil
}
