package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*PasswordResetToken)(nil)

type PasswordResetPurpose string

const (
	PasswordResetPurposeReset  PasswordResetPurpose = "reset"
	PasswordResetPurposeInvite PasswordResetPurpose = "invite"
)

func (p PasswordResetPurpose) IsInvite() bool {
	return p == PasswordResetPurposeInvite
}

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
	keel.Table    `keel:"password_reset_tokens"`

	ID             pulid.ID             `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:prt_"`
	UserID         pulid.ID             `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	BusinessUnitID pulid.ID             `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	OrganizationID pulid.ID             `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	TokenHash      string               `json:"-"              bun:"token_hash,type:VARCHAR(64),notnull" keel:"token_hash"`
	Purpose        PasswordResetPurpose `json:"purpose"        bun:"purpose,type:VARCHAR(20),notnull,default:'reset'" keel:"purpose"`
	ExpiresAt      int64                `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull" keel:"expires_at"`
	UsedAt         *int64               `json:"usedAt"         bun:"used_at,type:BIGINT,nullzero" keel:"used_at"`
	InvalidatedAt  *int64               `json:"invalidatedAt"  bun:"invalidated_at,type:BIGINT,nullzero" keel:"invalidated_at"`
	CreatedAt      int64                `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`

	User *User `json:"user,omitempty" bun:"rel:belongs-to,join:user_id=id" keel:"belongs:user_id"`
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
		if t.Purpose == "" {
			t.Purpose = PasswordResetPurposeReset
		}
	}
	return nil
}

func (t *PasswordResetToken) BeforeCreate(_ context.Context) error {
	if t.Purpose == "" {
		t.Purpose = PasswordResetPurposeReset
	}

	return nil
}
