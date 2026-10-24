package tenant

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

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
	keel.Table `keel:"password_reset_tokens"`

	ID             pulid.ID             `json:"id" keel:"id,pk,autoid:prt_"`
	UserID         pulid.ID             `json:"userId" keel:"user_id"`
	BusinessUnitID pulid.ID             `json:"businessUnitId" keel:"business_unit_id,tenant"`
	OrganizationID pulid.ID             `json:"organizationId" keel:"organization_id,tenant"`
	TokenHash      string               `json:"-" keel:"token_hash"`
	Purpose        PasswordResetPurpose `json:"purpose" keel:"purpose"`
	ExpiresAt      int64                `json:"expiresAt" keel:"expires_at"`
	UsedAt         *int64               `json:"usedAt" keel:"used_at"`
	InvalidatedAt  *int64               `json:"invalidatedAt" keel:"invalidated_at"`
	CreatedAt      int64                `json:"createdAt" keel:"created_at,autocreate"`

	User *User `json:"user,omitempty" keel:"belongs:user_id"`
}

// IsRedeemable reports whether the link still works. A token is good exactly once,
// and only before it expires.
func (t *PasswordResetToken) IsRedeemable(now int64) bool {
	return t.UsedAt == nil && t.InvalidatedAt == nil && now < t.ExpiresAt
}

func (t *PasswordResetToken) BeforeCreate(_ context.Context) error {
	if t.Purpose == "" {
		t.Purpose = PasswordResetPurposeReset
	}

	return nil
}
