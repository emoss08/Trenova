package tenant

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PasswordResetToken)(nil)

// PasswordResetToken is one outstanding "forgot password" link.
//
// The row is the entire record of a reset request: nothing on the user changes until
// a token is redeemed. That is what keeps an unauthenticated request from being a
// weapon — anyone can ask for a reset on any address, and the account owner's
// password keeps working until somebody who can actually read that mailbox uses the
// link.
//
// Only the SHA-256 digest of the token is stored, so a dump of this table yields
// nothing that can be redeemed.
type PasswordResetToken struct {
	bun.BaseModel `bun:"table:password_reset_tokens,alias:prt" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	TokenHash      string   `json:"-"              bun:"token_hash,type:VARCHAR(64),notnull"`
	ExpiresAt      int64    `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	UsedAt         *int64   `json:"usedAt"         bun:"used_at,type:BIGINT,nullzero"`
	InvalidatedAt  *int64   `json:"invalidatedAt"  bun:"invalidated_at,type:BIGINT,nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	User *User `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id"`
}

// IsRedeemable reports whether the link still works. A token is good exactly once,
// and only before it expires.
func (t *PasswordResetToken) IsRedeemable(now int64) bool {
	return t.UsedAt == nil && t.InvalidatedAt == nil && now < t.ExpiresAt
}

func (t *PasswordResetToken) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("prt_")
		}
		if t.CreatedAt == 0 {
			t.CreatedAt = timeutils.NowUnix()
		}
	}
	return nil
}
