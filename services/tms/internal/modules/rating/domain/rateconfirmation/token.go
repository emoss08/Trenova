package rateconfirmation

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateConfirmationToken)(nil)

type RateConfirmationToken struct {
	bun.BaseModel `bun:"table:rate_confirmation_tokens,alias:rct" json:"-"`
	keel.Table    `keel:"rate_confirmation_tokens"`

	ID                 pulid.ID `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:rct_"`
	BusinessUnitID     pulid.ID `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	RateConfirmationID pulid.ID `json:"rateConfirmationId" bun:"rate_confirmation_id,type:VARCHAR(100),notnull" keel:"rate_confirmation_id"`

	TokenHash string `json:"-"         bun:"token_hash,type:VARCHAR(128),notnull" keel:"token_hash"`
	Email     string `json:"email"     bun:"email,type:VARCHAR(255),notnull" keel:"email"`
	ExpiresAt int64  `json:"expiresAt" bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	UsedAt    *int64 `json:"usedAt"    bun:"used_at,type:BIGINT,nullzero" keel:"used_at"`
	RevokedAt *int64 `json:"revokedAt" bun:"revoked_at,type:BIGINT,nullzero" keel:"revoked_at"`
	CreatedAt int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	RateConfirmation *RateConfirmation `json:"rateConfirmation,omitempty" bun:"rel:belongs-to,join:rate_confirmation_id=id" keel:"belongs:rate_confirmation_id"`
}

func (rct *RateConfirmationToken) IsUsable(now int64) bool {
	return rct != nil &&
		rct.UsedAt == nil &&
		rct.RevokedAt == nil &&
		rct.ExpiresAt > now
}

func (rct *RateConfirmationToken) GetID() pulid.ID {
	return rct.ID
}

func (rct *RateConfirmationToken) GetCreatedAt() int64 {
	return rct.CreatedAt
}

func (rct *RateConfirmationToken) GetTableName() string {
	return "rate_confirmation_tokens"
}

func (rct *RateConfirmationToken) GetOrganizationID() pulid.ID {
	return rct.OrganizationID
}

func (rct *RateConfirmationToken) GetBusinessUnitID() pulid.ID {
	return rct.BusinessUnitID
}

func (rct *RateConfirmationToken) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if rct.ID.IsNil() {
			rct.ID = pulid.MustNew("rct_")
		}
		rct.CreatedAt = now
	case *bun.UpdateQuery:
		rct.UpdatedAt = now
	}

	return nil
}
