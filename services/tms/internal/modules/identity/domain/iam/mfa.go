package iam

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	RecoveryCodeHashSHA256 int16 = 1
	RecoveryCodeHashHMAC   int16 = 2
)

type MFARecoveryCode struct {
	keel.Table `keel:"mfa_recovery_codes"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:mrc_"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	CodeHash       string   `json:"-"              keel:"code_hash"`
	HashVersion    int16    `json:"-"              keel:"hash_version"`
	UsedAt         int64    `json:"usedAt"         keel:"used_at,nullzero"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

func (a *MFAAuthenticator) IsActiveTOTP() bool {
	return a != nil && a.Type == MFAAuthenticatorTypeTOTP && a.Enabled && a.VerifiedAt > 0
}
