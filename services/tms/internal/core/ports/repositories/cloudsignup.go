package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/cloudsignup"
	"github.com/emoss08/trenova/shared/pulid"
)

type RefreshCloudSignupRequest struct {
	ID           pulid.ID
	Name         string
	EmailAddress string
	CompanyName  string
	PasswordHash string
	TokenHash    string
	ExpiresAt    int64
	ClientIP     string
	UserAgent    string
}

type TouchCloudSignupRequest struct {
	ID        pulid.ID
	TokenHash string
	ExpiresAt int64
	ClientIP  string
	UserAgent string
}

type MarkCloudSignupProvisionedRequest struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	UserID         pulid.ID
	VerifiedAt     int64
}

type RejectCloudSignupRequest struct {
	ID     pulid.ID
	Reason string
}

type CloudSignupRepository interface {
	Create(ctx context.Context, entity *cloudsignup.CloudSignup) (*cloudsignup.CloudSignup, error)
	GetPendingByTokenHash(ctx context.Context, tokenHash string) (*cloudsignup.CloudSignup, error)
	GetPendingByEmail(
		ctx context.Context,
		emailNormalized string,
	) (*cloudsignup.CloudSignup, error)
	Refresh(
		ctx context.Context,
		req *RefreshCloudSignupRequest,
	) (*cloudsignup.CloudSignup, error)
	Touch(ctx context.Context, req *TouchCloudSignupRequest) (*cloudsignup.CloudSignup, error)
	IncrementAttempts(ctx context.Context, id pulid.ID) (int, error)
	MarkProvisioned(ctx context.Context, req *MarkCloudSignupProvisionedRequest) error
	Reject(ctx context.Context, req *RejectCloudSignupRequest) error
	Expire(ctx context.Context, now int64) (int64, error)
	CountProvisionedSince(ctx context.Context, since int64) (int, error)
}
