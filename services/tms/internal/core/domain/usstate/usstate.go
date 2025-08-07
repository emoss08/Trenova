/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package usstate

import (
	"context"

	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*UsState)(nil)

// UsState is the entity for a US state.
type UsState struct {
	bun.BaseModel `bun:"table:us_states,alias:ust" json:"-"`

	ID           pulid.ID `json:"id"           bun:"id,pk,type:VARCHAR(100)"`
	Name         string   `json:"name"         bun:"name,notnull"`
	Abbreviation string   `json:"abbreviation" bun:"abbreviation,notnull"`
	CountryName  string   `json:"countryName"  bun:"country_name,notnull"`
	CountryIso3  string   `json:"countryIso3"  bun:"country_iso3,notnull,default:'USA'"`
	CreatedAt    int64    `json:"createdAt"    bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64    `json:"updatedAt"    bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (us *UsState) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if us.ID.IsNil() {
			us.ID = pulid.MustNew("us_")
		}

		us.CreatedAt = now
	case *bun.UpdateQuery:
		us.UpdatedAt = now
	}

	return nil
}
