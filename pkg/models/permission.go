package models

import (
	"context"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Permission struct {
	bun.BaseModel    `bun:"permissions"`
	CreatedAt        time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt        time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID               uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Codename         string    `json:"codename" queryField:"true"`
	Action           string    `json:"action"`
	Label            string    `json:"label"`
	ReadDescription  string    `json:"readDescription"`
	WriteDescription string    `json:"writeDescription"`
	ResourceID       uuid.UUID `bun:"type:uuid" json:"resourceId"`

	Resource *Resource `bun:"rel:belongs-to,join:resource_id=id" json:"resource"`
}

func (p Permission) Validate() error {
	return validation.ValidateStruct(
		&p,
		validation.Field(&p.Codename, validation.Required),
	)
}

var _ bun.BeforeAppendModelHook = (*Permission)(nil)

func (p *Permission) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		p.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		p.UpdatedAt = time.Now()
	}
	return nil
}
