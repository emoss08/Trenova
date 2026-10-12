package iam

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*MFARecoveryCode)(nil)

type MFARecoveryCode struct {
	bun.BaseModel `bun:"table:mfa_recovery_codes,alias:mrc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	CodeHash       string   `json:"-"              bun:"code_hash,type:VARCHAR(64),notnull"`
	UsedAt         int64    `json:"usedAt"         bun:"used_at,nullzero"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (c *MFARecoveryCode) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &c.ID,
		IDPrefix:  "mrc_",
		CreatedAt: &c.CreatedAt,
	})
	return nil
}

func (a *MFAAuthenticator) IsActiveTOTP() bool {
	return a != nil && a.Type == MFAAuthenticatorTypeTOTP && a.Enabled && a.VerifiedAt > 0
}
