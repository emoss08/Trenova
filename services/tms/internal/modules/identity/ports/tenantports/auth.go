package tenantports

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type LoginRequest struct {
	EmailAddress     string `json:"emailAddress"               binding:"required,email"`
	Password         string `json:"password"                   binding:"required"`
	OrganizationSlug string `json:"organizationSlug,omitempty"`
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

type LoginResponse struct {
	User                   *tenant.User                  `json:"user"`
	ExpiresAt              int64                         `json:"expiresAt"`
	SessionID              string                        `json:"sessionId"`
	SessionToken           string                        `json:"-"`
	CSRFToken              string                        `json:"csrfToken,omitempty"`
	AuthProvider           string                        `json:"authProvider,omitempty"`
	ExternalIdentityID     string                        `json:"externalIdentityId,omitempty"`
	AuthenticatorAAL       int                           `json:"authenticatorAal"`
	FederationFAL          int                           `json:"federationFal"`
	MFAAuthenticatedAt     int64                         `json:"mfaAuthenticatedAt,omitempty"`
	LastReauthenticatedAt  int64                         `json:"lastReauthenticatedAt,omitempty"`
	RiskDecision           string                        `json:"riskDecision,omitempty"`
	ActiveRoleIDs          []pulid.ID                    `json:"activeRoleIds"`
	AuthorizedRoleIDs      []pulid.ID                    `json:"authorizedRoleIds"`
	ActiveRoles            []permissionports.RoleSummary `json:"activeRoles"`
	AuthorizedRoles        []permissionports.RoleSummary `json:"authorizedRoles"`
	RequiresRoleActivation bool                          `json:"requiresRoleActivation"`
	MFARequired            bool                          `json:"mfaRequired"`
	MFAChallengeToken      string                        `json:"mfaChallengeToken,omitempty"`
	MFAMethods             []string                      `json:"mfaMethods,omitempty"`
}

type VerifyMFAChallengeRequest struct {
	ChallengeToken string `json:"challengeToken"`
	Code           string `json:"code"`
	RecoveryCode   string `json:"recoveryCode"`
}

func (r *VerifyMFAChallengeRequest) Validate() error {
	me := errortypes.NewMultiError()
	if strings.TrimSpace(r.ChallengeToken) == "" {
		me.Add("challengeToken", errortypes.ErrRequired, "Sign in again to continue")
	}
	if strings.TrimSpace(r.Code) == "" && strings.TrimSpace(r.RecoveryCode) == "" {
		me.Add("code", errortypes.ErrRequired, "Enter the code from your authenticator app")
	}
	if me.HasErrors() {
		return me
	}

	return nil
}

type TenantLoginMetadataResponse struct {
	OrganizationID   string   `json:"organizationId"`
	OrganizationName string   `json:"organizationName"`
	OrganizationSlug string   `json:"organizationSlug"`
	EnabledProviders []string `json:"enabledProviders"`
	PasswordEnabled  bool     `json:"passwordEnabled"`
	EnforceSSO       bool     `json:"enforceSso"`
}

type AuthProviderSummary struct {
	ID       pulid.ID           `json:"id"`
	Name     string             `json:"name"`
	Provider tenant.SSOProvider `json:"provider"`
	Protocol tenant.SSOProtocol `json:"protocol"`
	Enabled  bool               `json:"enabled"`
}

type StartSSOLoginRequest struct {
	Provider         tenant.SSOProvider
	ProviderID       pulid.ID
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

type AuthenticatedPrincipal struct {
	Type           kernelports.PrincipalType
	PrincipalID    pulid.ID
	UserID         pulid.ID
	APIKeyID       pulid.ID
	BusinessUnitID pulid.ID
	OrganizationID pulid.ID
	APIKey         *apikey.Key
}

type ActivateSessionRolesRequest struct {
	SessionID pulid.ID   `json:"-"`
	RoleIDs   []pulid.ID `json:"roleIds"`
}

type ActivateSessionRolesResponse struct {
	ActiveRoleIDs          []pulid.ID                    `json:"activeRoleIds"`
	AuthorizedRoleIDs      []pulid.ID                    `json:"authorizedRoleIds"`
	ActiveRoles            []permissionports.RoleSummary `json:"activeRoles"`
	AuthorizedRoles        []permissionports.RoleSummary `json:"authorizedRoles"`
	RequiresRoleActivation bool                          `json:"requiresRoleActivation"`
}

type AuthorizedSessionRolesResponse struct {
	RoleIDs           []pulid.ID                    `json:"roleIds"`
	AuthorizedRoleIDs []pulid.ID                    `json:"authorizedRoleIds"`
	AuthorizedRoles   []permissionports.RoleSummary `json:"authorizedRoles"`
}

type AuthService interface {
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	GetTenantLoginMetadata(
		ctx context.Context,
		organizationSlug string,
	) (*TenantLoginMetadataResponse, error)
	ListAuthProviders(ctx context.Context, organizationSlug string) ([]AuthProviderSummary, error)
	StartSSOLogin(ctx context.Context, req StartSSOLoginRequest) (string, error)
	HandleSSOCallback(
		ctx context.Context,
		req SSOCallbackRequest,
	) (*SSOCallbackResponse, error)
	GetSSOLoginState(ctx context.Context, state string) (*SSOLoginState, error)
	ValidateSession(ctx context.Context, sessionID pulid.ID) (*session.Session, error)
	AuthenticateSession(ctx context.Context, token string) (*session.Session, error)
	KeepSessionAlive(ctx context.Context, token string) (*session.Session, error)
	ListAuthorizedSessionRoles(
		ctx context.Context,
		sessionID pulid.ID,
	) (*AuthorizedSessionRolesResponse, error)
	ActivateSessionRoles(
		ctx context.Context,
		req ActivateSessionRolesRequest,
	) (*ActivateSessionRolesResponse, error)
	AuthenticateAPIKey(
		ctx context.Context,
		token string,
		ipAddress, userAgent string,
	) (*AuthenticatedPrincipal, error)
	VerifyMFAChallenge(ctx context.Context, req VerifyMFAChallengeRequest) (*LoginResponse, error)
	Logout(ctx context.Context, sessionID pulid.ID) error
	CreateSessionForUser(
		ctx context.Context,
		req *CreateSessionForUserRequest,
	) (*LoginResponse, error)
}

type CreateSessionForUserRequest struct {
	User         *tenant.User
	AuthProvider string
}
