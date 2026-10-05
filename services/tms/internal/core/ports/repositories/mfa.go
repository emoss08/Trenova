package repositories

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/shared/pulid"
)

type ActivateTOTPRequest struct {
	AuthenticatorID pulid.ID
	UserID          pulid.ID
	Step            int64
	VerifiedAt      int64
	RecoveryCodes   []*iam.MFARecoveryCode
}

type RecordTOTPUseRequest struct {
	AuthenticatorID pulid.ID
	UserID          pulid.ID
	Step            int64
	UsedAt          int64
}

type ReplaceRecoveryCodesRequest struct {
	UserID pulid.ID
	Codes  []*iam.MFARecoveryCode
}

type ConsumeRecoveryCodeRequest struct {
	UserID   pulid.ID
	CodeHash string
	UsedAt   int64
}

type MFARepository interface {
	GetTOTP(ctx context.Context, userID pulid.ID) (*iam.MFAAuthenticator, error)
	ReplacePendingTOTP(ctx context.Context, authenticator *iam.MFAAuthenticator) error
	ActivateTOTP(ctx context.Context, req *ActivateTOTPRequest) error
	RecordTOTPUse(ctx context.Context, req RecordTOTPUseRequest) (bool, error)
	DeleteAll(ctx context.Context, userID pulid.ID) (int, error)
	ReplaceRecoveryCodes(ctx context.Context, req *ReplaceRecoveryCodesRequest) error
	ConsumeRecoveryCode(ctx context.Context, req ConsumeRecoveryCodeRequest) (bool, error)
	CountUnusedRecoveryCodes(ctx context.Context, userID pulid.ID) (int, error)
}

type MFAChallenge struct {
	UserID         pulid.ID `json:"userId"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	SwitchOrg      bool     `json:"switchOrg"`
	EmailAddress   string   `json:"emailAddress"`
	IssuedAt       int64    `json:"issuedAt"`
	ExpiresAt      int64    `json:"expiresAt"`
}

type MFAChallengeRepository interface {
	Save(ctx context.Context, tokenHash string, challenge *MFAChallenge, ttl time.Duration) error
	Get(ctx context.Context, tokenHash string) (*MFAChallenge, error)
	Delete(ctx context.Context, tokenHash string) error
	RecordFailure(ctx context.Context, tokenHash string, ttl time.Duration) (int64, error)
}
