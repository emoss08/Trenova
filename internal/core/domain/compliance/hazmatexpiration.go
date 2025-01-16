package compliance

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/usstate"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*HazmatExpiration)(nil)

// HazmatExpiration is a model that stores information about hazmat expirations
type HazmatExpiration struct {
	bun.BaseModel `bun:"table:hazmat_expirations,alias:he" json:"-"`

	// Primary identifiers
	ID      pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	StateID pulid.ID `bun:"state_id,type:VARCHAR(100),pk,notnull" json:"stateId"`

	// Core fields
	Years int8 `bun:"years,type:SMALLINT,notnull" json:"years"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT,notnull" json:"version"`
	CreatedAt int64 `bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

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
