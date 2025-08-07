/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package compliance

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*HazmatExpiration)(nil)

// HazmatExpiration is a model that stores information about hazmat expirations
type HazmatExpiration struct {
	bun.BaseModel `bun:"table:hazmat_expirations,alias:he" json:"-"`

	ID        pulid.ID `json:"id"        bun:"id,type:VARCHAR(100),pk,notnull"`
	StateID   pulid.ID `json:"stateId"   bun:"state_id,type:VARCHAR(100),pk,notnull"`
	Years     int8     `json:"years"     bun:"years,type:SMALLINT,notnull"`
	Version   int64    `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	State *usstate.UsState `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id"`
}

// BeforeAppendModel is a bun hook that sets the createdAt and updatedAt fields
func (he *HazmatExpiration) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if he.ID == "" {
			he.ID = pulid.MustNew("he_")
		}

		he.CreatedAt = now
	case *bun.UpdateQuery:
		he.UpdatedAt = now
	}

	return nil
}
