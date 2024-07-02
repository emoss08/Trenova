package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Resource struct {
	bun.BaseModel `bun:"resources"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Type          string    `json:"type" queryField:"true"`
	Description   string    `json:"description"`
}

var _ bun.BeforeAppendModelHook = (*Resource)(nil)

func (c *Resource) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
