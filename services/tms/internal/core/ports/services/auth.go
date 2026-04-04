package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type LoginRequest struct {
	EmailAddress     string `json:"emailAddress" binding:"required,email"`
	Password         string `json:"password"     binding:"required"`
	OrganizationSlug string `json:"organizationSlug,omitempty"`
}

type LoginResponse struct {
	User      *tenant.User `json:"user"`
	ExpiresAt int64        `json:"expiresAt"`
	SessionID string       `json:"sessionId"`
}

type TenantLoginMetadataResponse struct {
	OrganizationID   string   `json:"organizationId"`
	OrganizationName string   `json:"organizationName"`
	OrganizationSlug string   `json:"organizationSlug"`
	EnabledProviders []string `json:"enabledProviders"`
	PasswordEnabled  bool     `json:"passwordEnabled"`
	EnforceSSO       bool     `json:"enforceSso"`
}

type StartSSOLoginRequest struct {
	Provider         tenant.SSOProvider
	OrganizationSlug string
	ReturnTo         string
}

type SSOCallbackRequest struct {
	State string
	Code  string
}

type SSOCallbackResponse struct {
	LoginResponse *LoginResponse
	RedirectTo    string
}

type PrincipalType string

const (
	PrincipalTypeUser   PrincipalType = "session_user"
	PrincipalTypeAPIKey PrincipalType = "api_key"
)

type AuthenticatedPrincipal struct {
	Type           PrincipalType
	PrincipalID    pulid.ID
	UserID         pulid.ID
	APIKeyID       pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	APIKey         *apikey.Key
}

type RequestActor struct {
	PrincipalType  PrincipalType `json:"principalType"`
	PrincipalID    pulid.ID      `json:"principalId"`
	UserID         pulid.ID      `json:"userId,omitempty"`
	APIKeyID       pulid.ID      `json:"apiKeyId,omitempty"`
	BusinessUnitID pulid.ID      `json:"businessUnitId"`
	OrganizationID pulid.ID      `json:"organizationId"`
}

func (a *RequestActor) IsAPIKey() bool {
	if a == nil {
		return false
	}
	return a.PrincipalType == PrincipalTypeAPIKey
}

func (a *RequestActor) IsUser() bool {
	if a == nil {
		return false
	}
	return a.PrincipalType == PrincipalTypeUser
}

func (lr *LoginRequest) Validate() error {
	me := errortypes.NewMultiError()

	valErr := validation.ValidateStruct(
		lr,
		validation.Field(
			&lr.EmailAddress,
			validation.Length(1, 255).Error("Email address must be between 1 and 255 characters"),
			is.EmailFormat.Error("Email address must be a valid email address"),
			validation.Required.Error(
				"Email address is required. Please provide a valid email address.",
			),
		),
		validation.Field(
			&lr.Password,
			validation.Required.Error("Password is required. Please provide a valid password."),
		),
	)

	me.AddOzzoError(valErr)

	if me.HasErrors() {
		return me
	}

	return nil
}

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	GetTenantLoginMetadata(
		ctx context.Context,
		organizationSlug string,
	) (*TenantLoginMetadataResponse, error)
	StartSSOLogin(ctx context.Context, req StartSSOLoginRequest) (string, error)
	HandleSSOCallback(
		ctx context.Context,
		req SSOCallbackRequest,
	) (*SSOCallbackResponse, error)
	GetSSOLoginState(ctx context.Context, state string) (*repositories.SSOLoginState, error)
	ValidateSession(ctx context.Context, sessionID pulid.ID) (*session.Session, error)
	AuthenticateAPIKey(
		ctx context.Context,
		token string,
		ipAddress, userAgent string,
	) (*AuthenticatedPrincipal, error)
	Logout(ctx context.Context, sessionID pulid.ID) error
}
