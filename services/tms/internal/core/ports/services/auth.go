package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type LoginRequest struct {
	EmailAddress string `json:"emailAddress"`
	Password     string `json:"password"`
	ClientIP     string `json:"clientIp"`
	UserAgent    string `json:"userAgent"`
}

func (lr *LoginRequest) Validate() error {
	return validation.ValidateStruct(lr,
		validation.
			Field(
				&lr.EmailAddress,
				validation.Required.Error("Email address is required. Please try again."),
				validation.Length(1, 255).
					Error("Email address must be between 1 and 255 characters. Please try again."),
			),
		validation.
			Field(&lr.Password,
				validation.Required.Error("Password is required. Please try again."),
			),
	)
}

type LoginResponse struct {
	User      *tenant.User `json:"user"`
	ExpiresAt int64        `json:"expiresAt"`
	SessionID string       `json:"sessionId"`
}

type ValidateSessionRequest struct {
	SessionID pulid.ID
	ClientIP  string
}

type RefreshSessionRequest struct {
	SessionID pulid.ID
	ClientIP  string
	UserAgent string
	Metadata  map[string]any
}
type LogoutRequest struct {
	SessionID pulid.ID
	ClientIP  string
	UserAgent string
	Reason    string // Optional reason for logout
}

type CheckEmailRequest struct {
	EmailAddress string `json:"emailAddress"`
}

func (r *CheckEmailRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(
			&r.EmailAddress,
			validation.Required.Error("Email address is required"),
			is.EmailFormat.Error("Invalid email format. Please try again"),
		),
	)
}

type CreateAPITokenRequest struct {
	UserID         pulid.ID               `json:"userId"`
	BusinessUnitID pulid.ID               `json:"businessUnitId"`
	OrganizationID pulid.ID               `json:"organizationId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Scopes         []tenant.APITokenScope `json:"scopes"`
	ExpiresAt      *int64                 `json:"expiresAt,omitempty"`
}

func (r *CreateAPITokenRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.Name, validation.Required, validation.Length(1, 255)),
		validation.Field(&r.Scopes, validation.Required, validation.Length(1, 100)),
	)
}

type CreateAPITokenResponse struct {
	Token      *tenant.APIToken `json:"token"`
	PlainToken string           `json:"plainToken"` // Only returned on creation
}

type ValidateAPITokenRequest struct {
	Token    string `json:"token"`
	ClientIP string `json:"clientIp"`
}

type RevokeAPITokenRequest struct {
	TokenID pulid.ID `json:"tokenId"`
	UserID  pulid.ID `json:"userId"` // For authorization check
}

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	ValidateSession(ctx context.Context, req ValidateSessionRequest) (bool, error)
	RefreshSession(
		ctx context.Context,
		req RefreshSessionRequest,
	) (*session.Session, error)
	Logout(ctx context.Context, req LogoutRequest) error
	UpdateSessionOrganization(ctx context.Context, sessionID pulid.ID, newOrgID pulid.ID) error
	CheckEmail(ctx context.Context, req CheckEmailRequest) (bool, error)
	CreateAPIToken(ctx context.Context, req *CreateAPITokenRequest) (*CreateAPITokenResponse, error)
	ValidateAPIToken(ctx context.Context, req ValidateAPITokenRequest) (*tenant.APIToken, error)
	RevokeAPIToken(ctx context.Context, req RevokeAPITokenRequest) error
	ListUserAPITokens(ctx context.Context, userID pulid.ID) ([]*tenant.APIToken, error)
}
