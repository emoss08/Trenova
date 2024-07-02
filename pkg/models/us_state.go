package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UsState struct {
	bun.BaseModel `bun:"table:us_states,alias:us" json:"-"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID            uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Name          string    `bun:",notnull" json:"name"`
	Abbreviation  string    `bun:",notnull" json:"abbreviation"`
	CountryName   string    `bun:",notnull" json:"countryName"`
	CountryIso3   string    `bun:",notnull,default:'USA'" json:"countryIso3"`
}

var _ bun.BeforeAppendModelHook = (*UsState)(nil)

func (c *UsState) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
