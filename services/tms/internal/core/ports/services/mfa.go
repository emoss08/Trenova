package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MFAMethodTOTP         = "totp"
	MFAMethodRecoveryCode = "recovery_code"
)

type MFAStatus struct {
	TOTPEnabled            bool  `json:"totpEnabled"`
	EnrollmentPending      bool  `json:"enrollmentPending"`
	EnabledAt              int64 `json:"enabledAt,omitempty"`
	LastUsedAt             int64 `json:"lastUsedAt,omitempty"`
	RecoveryCodesRemaining int   `json:"recoveryCodesRemaining"`
	SessionVerified        bool  `json:"sessionVerified"`
}

type BeginTOTPEnrollmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Password   string                `json:"password"`
}

type TOTPEnrollment struct {
	AuthenticatorID pulid.ID `json:"authenticatorId"`
	Secret          string   `json:"secret"`
	URI             string   `json:"uri"`
	QRCode          string   `json:"qrCode"`
}

type ConfirmTOTPEnrollmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	SessionID  pulid.ID              `json:"-"`
	Code       string                `json:"code"`
	Name       string                `json:"name"`
}

type RecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recoveryCodes"`
}

type DisableTOTPRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	Password     string                `json:"password"`
	Code         string                `json:"code"`
	RecoveryCode string                `json:"recoveryCode"`
}

type RegenerateRecoveryCodesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Code       string                `json:"code"`
}

type ResetUserMFARequest struct {
	TenantInfo   pagination.TenantInfo
	TargetUserID pulid.ID
}

type VerifySecondFactorRequest struct {
	UserID       pulid.ID
	Code         string
	RecoveryCode string
}

type ReauthenticateRequest struct {
	UserID       pulid.ID
	Password     string
	Code         string
	RecoveryCode string
}

type MFAService interface {
	Status(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		sessionAAL int,
	) (*MFAStatus, error)
	BeginTOTPEnrollment(
		ctx context.Context,
		req *BeginTOTPEnrollmentRequest,
	) (*TOTPEnrollment, error)
	ConfirmTOTPEnrollment(
		ctx context.Context,
		req *ConfirmTOTPEnrollmentRequest,
	) (*RecoveryCodesResponse, error)
	DisableTOTP(ctx context.Context, req *DisableTOTPRequest) error
	RegenerateRecoveryCodes(
		ctx context.Context,
		req *RegenerateRecoveryCodesRequest,
	) (*RecoveryCodesResponse, error)
	ResetUserMFA(ctx context.Context, req *ResetUserMFARequest) error
	HasActiveFactor(ctx context.Context, userID pulid.ID) (bool, error)
	VerifySecondFactor(ctx context.Context, req *VerifySecondFactorRequest) (string, error)
	Reauthenticate(ctx context.Context, req *ReauthenticateRequest) (string, error)
}
