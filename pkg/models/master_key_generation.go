package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type MasterKeyGeneration struct {
	bun.BaseModel  `bun:"table:master_key_generations,alias:mkg" json:"-"`
	CreatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

var _ bun.BeforeAppendModelHook = (*MasterKeyGeneration)(nil)

func (m *MasterKeyGeneration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		m.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		m.UpdatedAt = time.Now()
	}
	return nil
}
