//nolint:funlen,gocritic // existing legacy workflow/API shape is intentionally kept stable
package authservice

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Params struct {
	fx.In

	UserRepository    repositories.UserRepository
	OrganizationRepo  repositories.OrganizationRepository
	SessionRepository repositories.SessionRepository
	RBACRepository    repositories.RBACRepository
	SSOConfigRepo     repositories.SSOConfigRepository
	SSOStateRepo      repositories.SSOLoginStateRepository
	APIKeyRepository  repositories.APIKeyRepository
	UsageRecorder     services.UsageRecorder
	Encryption        *encryptionservice.Service
	Config            *config.Config
	Logger            *zap.Logger
}

type Service struct {
	ur        repositories.UserRepository
	or        repositories.OrganizationRepository
	sr        repositories.SessionRepository
	rbacRepo  repositories.RBACRepository
	ssoRepo   repositories.SSOConfigRepository
	stateRepo repositories.SSOLoginStateRepository
	akr       repositories.APIKeyRepository
	usageBuf  services.UsageRecorder
	enc       *encryptionservice.Service
	cfg       *config.Config
	l         *zap.Logger
}

func New(p Params) services.AuthService {
	return &Service{
		ur:        p.UserRepository,
		or:        p.OrganizationRepo,
		sr:        p.SessionRepository,
		rbacRepo:  p.RBACRepository,
		ssoRepo:   p.SSOConfigRepo,
		stateRepo: p.SSOStateRepo,
		akr:       p.APIKeyRepository,
		usageBuf:  p.UsageRecorder,
		enc:       p.Encryption,
		cfg:       p.Config,
		l:         p.Logger.Named("service.auth"),
	}
}

var errInvalidCredentials = errortypes.NewAuthenticationError("Invalid email or password")
var errSSORequired = errortypes.NewAuthenticationError(
	"Password login is disabled for this organization. Use SSO to sign in.",
)

const ssoLoginStateTTL = 10 * time.Minute

type tenantSSOConfiguration struct {
	organization *tenant.Organization
	configs      []*tenant.SSOConfig
}

type loginSessionContext struct {
	AuthProvider          string
	ExternalIdentityID    string
	ExternalSubject       string
	AuthenticatorAAL      int
	FederationFAL         int
	MFAAuthenticatedAt    int64
	LastReauthenticatedAt int64
	RiskDecision          string
	RiskDecisionID        pulid.ID
}

func (s *Service) Login(
	ctx context.Context,
	req services.LoginRequest,
) (*services.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	usr, err := s.ur.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return nil, errInvalidCredentials
	}

	if err = usr.VerifyCredentials(req.Password); err != nil {
		if errortypes.IsAuthorizationError(err) {
			return nil, err
		}
		return nil, errInvalidCredentials
	}

	targetOrg, err := s.resolveRequestedOrganization(ctx, req.OrganizationSlug, usr)
	if err != nil {
		return nil, err
	}

	if err = s.enforcePasswordLoginPolicy(ctx, req.OrganizationSlug, usr, targetOrg); err != nil {
		return nil, err
	}

	if targetOrg != nil && targetOrg.ID != usr.CurrentOrganizationID {
		if err = s.ur.UpdateCurrentOrganization(
			ctx,
			usr.ID,
			targetOrg.ID,
			targetOrg.BusinessUnitID,
		); err != nil {
			return nil, err
		}

		usr.CurrentOrganizationID = targetOrg.ID
		usr.BusinessUnitID = targetOrg.BusinessUnitID
	}

	return s.createLoginResponse(ctx, usr, loginSessionContext{
		AuthProvider:          "password",
		AuthenticatorAAL:      1,
		FederationFAL:         1,
		LastReauthenticatedAt: timeutils.NowUnix(),
		RiskDecision:          "allow",
	})
}

func (s *Service) GetTenantLoginMetadata(
	ctx context.Context,
	organizationSlug string,
) (*services.TenantLoginMetadataResponse, error) {
	loginConfig, err := s.enabledTenantSSOConfiguration(ctx, organizationSlug)
	if err != nil {
		return nil, err
	}

	org := loginConfig.organization
	resp := &services.TenantLoginMetadataResponse{
		OrganizationID:   org.ID.String(),
		OrganizationName: org.Name,
		OrganizationSlug: org.LoginSlug,
		PasswordEnabled:  true,
	}

	var anyEnforced bool
	enabledProviders := make([]string, 0, len(loginConfig.configs))
	for _, cfg := range loginConfig.configs {
		enabledProviders = append(enabledProviders, string(cfg.Provider))
		if cfg.EnforceSSO {
			anyEnforced = true
		}
	}
	resp.EnabledProviders = enabledProviders
	resp.EnforceSSO = anyEnforced
	resp.PasswordEnabled = !anyEnforced

	return resp, nil
}

func (s *Service) ListAuthProviders(
	ctx context.Context,
	organizationSlug string,
) ([]services.AuthProviderSummary, error) {
	loginConfig, err := s.enabledTenantSSOConfiguration(ctx, organizationSlug)
	if err != nil {
		return nil, err
	}

	providers := make([]services.AuthProviderSummary, 0, len(loginConfig.configs))
	for _, cfg := range loginConfig.configs {
		providers = append(providers, services.AuthProviderSummary{
			ID:       cfg.ID,
			Name:     cfg.Name,
			Provider: cfg.Provider,
			Protocol: cfg.Protocol,
			Enabled:  cfg.Enabled,
		})
	}

	return providers, nil
}

func (s *Service) StartSSOLogin(
	ctx context.Context,
	req services.StartSSOLoginRequest,
) (string, error) {
	if s.or == nil || s.ssoRepo == nil || s.stateRepo == nil {
		return "", errortypes.NewBusinessError(
			providerDisplayName(req.Provider) + " SSO is not configured",
		)
	}

	org, err := s.or.GetByLoginSlug(ctx, req.OrganizationSlug)
	if err != nil {
		return "", err
	}

	ssoConfig, err := s.resolveSSOConfig(ctx, org.ID, req)
	if err != nil {
		return "", err
	}
	req.Provider = ssoConfig.Provider

	if err = validateReturnToURL(req.ReturnTo, s.cfg.Server.CORS.AllowedOrigins); err != nil {
		return "", err
	}

	provider, err := oidc.NewProvider(ctx, ssoConfig.OIDCIssuerURL)
	if err != nil {
		return "", errortypes.NewBusinessError("Failed to initialize " + providerDisplayName(req.Provider) + " identity provider").
			WithInternal(err)
	}

	oauthCfg := oauth2.Config{
		ClientID:     ssoConfig.OIDCClientID,
		ClientSecret: decryptSSOSecret(s.enc, ssoConfig),
		Endpoint:     provider.Endpoint(),
		RedirectURL:  ssoConfig.OIDCRedirectURL,
		Scopes:       ssoConfig.OIDCScopes,
	}

	state := randomURLToken(32)
	nonce := randomURLToken(32)
	verifier := oauth2.GenerateVerifier()

	if err = s.stateRepo.Save(ctx, &repositories.SSOLoginState{
		State:            state,
		Provider:         req.Provider,
		ProviderID:       ssoConfig.ID,
		OrganizationID:   org.ID,
		OrganizationSlug: org.LoginSlug,
		CodeVerifier:     verifier,
		Nonce:            nonce,
		ReturnTo:         req.ReturnTo,
	}, ssoLoginStateTTL); err != nil {
		return "", err
	}

	return oauthCfg.AuthCodeURL(
		state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

func (s *Service) resolveSSOConfig(
	ctx context.Context,
	orgID pulid.ID,
	req services.StartSSOLoginRequest,
) (*tenant.SSOConfig, error) {
	if req.ProviderID.IsNotNil() {
		cfg, err := s.ssoRepo.GetEnabledByID(ctx, req.ProviderID)
		if err != nil {
			return nil, err
		}
		if cfg.OrganizationID != orgID {
			return nil, errortypes.NewAuthenticationError(
				"SSO provider does not belong to this organization",
			)
		}
		return cfg, nil
	}

	return s.ssoRepo.GetEnabledByOrganizationID(ctx, orgID, req.Provider)
}

func (s *Service) HandleSSOCallback( //nolint:cyclop // legacy workflow
	ctx context.Context,
	req services.SSOCallbackRequest,
) (*services.SSOCallbackResponse, error) {
	if s.or == nil || s.ssoRepo == nil || s.stateRepo == nil {
		return nil, errortypes.NewBusinessError("SSO is not configured")
	}

	loginState, err := s.stateRepo.Get(ctx, req.State)
	if err != nil {
		return nil, errortypes.NewAuthenticationError("SSO login session is invalid or expired")
	}
	defer func() {
		if delErr := s.stateRepo.Delete(ctx, req.State); delErr != nil {
			s.l.Warn("failed to delete sso login state", zap.Error(delErr))
		}
	}()

	displayName := providerDisplayName(loginState.Provider)

	ssoConfig, err := s.resolveSSOConfigForCallback(ctx, loginState)
	if err != nil {
		return nil, err
	}

	provider, err := oidc.NewProvider(ctx, ssoConfig.OIDCIssuerURL)
	if err != nil {
		return nil, errortypes.NewBusinessError("Failed to initialize " + displayName + " identity provider").
			WithInternal(err)
	}

	oauthCfg := oauth2.Config{
		ClientID:     ssoConfig.OIDCClientID,
		ClientSecret: decryptSSOSecret(s.enc, ssoConfig),
		Endpoint:     provider.Endpoint(),
		RedirectURL:  ssoConfig.OIDCRedirectURL,
		Scopes:       ssoConfig.OIDCScopes,
	}

	oauthToken, err := oauthCfg.Exchange(
		ctx,
		req.Code,
		oauth2.VerifierOption(loginState.CodeVerifier),
	)
	if err != nil {
		return nil, errortypes.NewAuthenticationError(displayName + " login failed")
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return nil, errortypes.NewAuthenticationError(
			displayName + " login did not return an ID token",
		)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: ssoConfig.OIDCClientID,
	})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, errortypes.NewAuthenticationError(displayName + " identity token is invalid")
	}

	var claims oidcClaims
	if err = idToken.Claims(&claims); err != nil {
		return nil, errortypes.NewAuthenticationError(displayName + " identity token is invalid")
	}

	if claims.Nonce != loginState.Nonce {
		return nil, errortypes.NewAuthenticationError(displayName + " login nonce mismatch")
	}

	if ssoConfig.Provider == tenant.SSOProviderAzureAD {
		expectedTenantID := strings.TrimSpace(microsoftTenantIDFromIssuer(ssoConfig.OIDCIssuerURL))
		if expectedTenantID != "" && !strings.EqualFold(expectedTenantID, claims.TenantID) {
			return nil, errortypes.NewAuthenticationError(
				"Microsoft tenant does not match this organization's configuration",
			)
		}
	}

	emailAddress := claims.EmailAddress()
	if emailAddress == "" {
		return nil, errortypes.NewAuthenticationError(
			displayName + " account did not provide a usable email address",
		)
	}

	if err = validateAllowedDomain(emailAddress, ssoConfig.AllowedDomains); err != nil {
		return nil, err
	}

	usr, err := s.ur.FindByEmail(ctx, emailAddress)
	if err != nil {
		return nil, errortypes.NewAuthenticationError(
			"No Trenova user exists for this " + displayName + " account",
		)
	}

	if err = usr.ValidateStatus(); err != nil {
		return nil, err
	}

	if err = s.ensureUserHasOrganizationAccess(ctx, usr.ID, loginState.OrganizationID); err != nil {
		return nil, err
	}

	if loginState.OrganizationID != usr.CurrentOrganizationID {
		if err = s.ur.UpdateCurrentOrganization(
			ctx,
			usr.ID,
			loginState.OrganizationID,
			ssoConfig.BusinessUnitID,
		); err != nil {
			return nil, err
		}
		usr.CurrentOrganizationID = loginState.OrganizationID
		usr.BusinessUnitID = ssoConfig.BusinessUnitID
	}

	aal, mfaAt := assuranceFromOIDCClaims(claims)
	loginResp, err := s.createLoginResponse(ctx, usr, loginSessionContext{
		AuthProvider:          string(loginState.Provider),
		ExternalSubject:       claims.Subject,
		AuthenticatorAAL:      aal,
		FederationFAL:         2,
		MFAAuthenticatedAt:    mfaAt,
		LastReauthenticatedAt: timeutils.NowUnix(),
		RiskDecision:          "allow",
	})
	if err != nil {
		return nil, err
	}

	return &services.SSOCallbackResponse{
		LoginResponse: loginResp,
		RedirectTo:    loginState.ReturnTo,
	}, nil
}

func (s *Service) GetSSOLoginState(
	ctx context.Context,
	state string,
) (*repositories.SSOLoginState, error) {
	if s.stateRepo == nil {
		return nil, errortypes.NewBusinessError("SSO is not configured")
	}
	return s.stateRepo.Get(ctx, state)
}

func (s *Service) enabledTenantSSOConfiguration(
	ctx context.Context,
	organizationSlug string,
) (*tenantSSOConfiguration, error) {
	if s.or == nil || s.ssoRepo == nil {
		return nil, errortypes.NewBusinessError("Tenant login is not configured")
	}

	org, err := s.or.GetByLoginSlug(ctx, organizationSlug)
	if err != nil {
		return nil, err
	}

	configs, err := s.ssoRepo.ListEnabledByOrganizationID(ctx, org.ID)
	if err != nil {
		return nil, err
	}

	return &tenantSSOConfiguration{
		organization: org,
		configs:      configs,
	}, nil
}

func (s *Service) resolveSSOConfigForCallback(
	ctx context.Context,
	loginState *repositories.SSOLoginState,
) (*tenant.SSOConfig, error) {
	if loginState.ProviderID.IsNotNil() {
		cfg, err := s.ssoRepo.GetEnabledByID(ctx, loginState.ProviderID)
		if err != nil {
			return nil, err
		}
		if cfg.OrganizationID != loginState.OrganizationID {
			return nil, errortypes.NewAuthenticationError(
				"SSO provider does not belong to this organization",
			)
		}
		return cfg, nil
	}

	return s.ssoRepo.GetEnabledByOrganizationID(
		ctx,
		loginState.OrganizationID,
		loginState.Provider,
	)
}

func (s *Service) ValidateSession(
	ctx context.Context,
	sessionID pulid.ID,
) (*session.Session, error) {
	sess, err := s.sr.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err = sess.Validate(); err != nil {
		if delErr := s.sr.Delete(ctx, sessionID); delErr != nil {
			s.l.Error("failed to delete expired session", zap.Error(delErr))
		}
		return nil, errortypes.NewAuthenticationError("Session has expired. Please login again.")
	}

	return sess, nil
}

func (s *Service) ListAuthorizedSessionRoles(
	ctx context.Context,
	sessionID pulid.ID,
) (*services.AuthorizedSessionRolesResponse, error) {
	sess, err := s.ValidateSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	authorizedRoles, err := s.authorizedRoleSummaries(ctx, sess.UserID, sess.OrganizationID)
	if err != nil {
		return nil, err
	}

	authorizedRoleIDs := roleSummaryIDs(authorizedRoles)
	return &services.AuthorizedSessionRolesResponse{
		RoleIDs:           authorizedRoleIDs,
		AuthorizedRoleIDs: authorizedRoleIDs,
		AuthorizedRoles:   authorizedRoles,
	}, nil
}

func (s *Service) ActivateSessionRoles(
	ctx context.Context,
	req services.ActivateSessionRolesRequest,
) (*services.ActivateSessionRolesResponse, error) {
	sess, err := s.ValidateSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	authorizedRoles, err := s.authorizedRoleSummaries(ctx, sess.UserID, sess.OrganizationID)
	if err != nil {
		return nil, err
	}
	authorizedRoleIDs := roleSummaryIDs(authorizedRoles)
	authorized := make(map[pulid.ID]struct{}, len(authorizedRoleIDs))
	for _, id := range authorizedRoleIDs {
		authorized[id] = struct{}{}
	}

	activeRoleIDs := make([]pulid.ID, 0, len(req.RoleIDs))
	seen := make(map[pulid.ID]struct{}, len(req.RoleIDs))
	for _, roleID := range req.RoleIDs {
		if roleID.IsNil() {
			continue
		}
		if _, ok := seen[roleID]; ok {
			continue
		}
		if _, ok := authorized[roleID]; !ok {
			return nil, errortypes.NewAuthorizationError("Cannot activate unauthorized role")
		}
		seen[roleID] = struct{}{}
		activeRoleIDs = append(activeRoleIDs, roleID)
	}
	if len(activeRoleIDs) == 0 && len(authorizedRoleIDs) > 0 {
		return nil, errortypes.NewValidationError(
			"roleIds",
			errortypes.ErrRequired,
			"At least one active role is required",
		)
	}

	violations, vErr := s.rbacRepo.ValidateDynamicSeparationOfDuty(
		ctx,
		sess.OrganizationID,
		activeRoleIDs,
	)
	if vErr != nil {
		return nil, vErr
	}
	if len(violations) > 0 {
		return nil, errortypes.NewAuthorizationError(
			"Selected active roles violate dynamic separation of duty",
		)
	}

	sess.ActiveRoleIDs = activeRoleIDs
	if err = s.sr.Update(ctx, sess); err != nil {
		return nil, err
	}

	return &services.ActivateSessionRolesResponse{
		ActiveRoleIDs:          activeRoleIDs,
		AuthorizedRoleIDs:      authorizedRoleIDs,
		ActiveRoles:            activeRoleSummaries(activeRoleIDs, authorizedRoles),
		AuthorizedRoles:        authorizedRoles,
		RequiresRoleActivation: len(activeRoleIDs) == 0 && len(authorizedRoleIDs) > 0,
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID pulid.ID) error {
	return s.sr.Delete(ctx, sessionID)
}

func (s *Service) AuthenticateAPIKey(
	ctx context.Context,
	token string,
	ipAddress, userAgent string,
) (*services.AuthenticatedPrincipal, error) {
	if !s.cfg.Security.APIToken.Enabled {
		return nil, errortypes.NewAuthenticationError("API token authentication is disabled")
	}

	prefix, secret, err := apikey.SplitToken(token)
	if err != nil {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	key, err := s.akr.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	if key.Status != apikey.StatusActive {
		return nil, errortypes.NewAuthenticationError("API key is inactive")
	}

	if key.IsExpired(timeutils.NowUnix()) {
		return nil, errortypes.NewAuthenticationError("API key has expired")
	}

	computed := apikey.HashSecret(key.SecretSalt, secret)
	if subtle.ConstantTimeCompare([]byte(computed), []byte(key.SecretHash)) != 1 {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	s.usageBuf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       key.ID,
		OrganizationID: key.OrganizationID,
		BusinessUnitID: key.BusinessUnitID,
		OccurredAt:     time.Now().UTC(),
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	})

	return &services.AuthenticatedPrincipal{
		Type:           services.PrincipalTypeAPIKey,
		PrincipalID:    key.ID,
		APIKeyID:       key.ID,
		BusinessUnitID: key.BusinessUnitID,
		OrganizationID: key.OrganizationID,
		APIKey:         key,
	}, nil
}

func (s *Service) createSession(
	ctx context.Context,
	user *tenant.User,
	authn loginSessionContext,
) (*session.Session, error) {
	expiresAt := timeutils.NowUnix() + int64(session.DefaultTTL.Seconds())

	sess := session.NewSession(&session.NewSessionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  user.CurrentOrganizationID,
			BuID:   user.BusinessUnitID,
			UserID: user.ID,
		},
		ExpiresAt:             expiresAt,
		AuthProvider:          authn.AuthProvider,
		ExternalIdentityID:    authn.ExternalIdentityID,
		ExternalSubject:       authn.ExternalSubject,
		AuthenticatorAAL:      authn.AuthenticatorAAL,
		FederationFAL:         authn.FederationFAL,
		MFAAuthenticatedAt:    authn.MFAAuthenticatedAt,
		LastReauthenticatedAt: authn.LastReauthenticatedAt,
		RiskDecision:          authn.RiskDecision,
		RiskDecisionID:        authn.RiskDecisionID,
	})

	if err := s.sr.Create(ctx, sess); err != nil {
		return nil, err
	}

	return sess, nil
}

func (s *Service) createLoginResponse(
	ctx context.Context,
	user *tenant.User,
	authn loginSessionContext,
) (*services.LoginResponse, error) {
	sess, err := s.createSession(ctx, user, authn)
	if err != nil {
		return nil, err
	}

	if err = s.ur.UpdateLastLoginAt(ctx, user.ID); err != nil {
		s.l.Error("failed to update last login at", zap.Error(err))
	}

	authorizedRoles, err := s.authorizedRoleSummaries(ctx, user.ID, user.CurrentOrganizationID)
	if err != nil {
		return nil, err
	}
	authorizedRoleIDs := roleSummaryIDs(authorizedRoles)

	return &services.LoginResponse{
		User:                  user,
		ExpiresAt:             sess.ExpiresAt,
		SessionID:             sess.ID.String(),
		AuthProvider:          sess.AuthProvider,
		ExternalIdentityID:    sess.ExternalIdentityID,
		AuthenticatorAAL:      sess.AuthenticatorAAL,
		FederationFAL:         sess.FederationFAL,
		MFAAuthenticatedAt:    sess.MFAAuthenticatedAt,
		LastReauthenticatedAt: sess.LastReauthenticatedAt,
		RiskDecision:          sess.RiskDecision,
		ActiveRoleIDs:         sess.ActiveRoleIDs,
		AuthorizedRoleIDs:     authorizedRoleIDs,
		ActiveRoles:           activeRoleSummaries(sess.ActiveRoleIDs, authorizedRoles),
		AuthorizedRoles:       authorizedRoles,
		RequiresRoleActivation: len(sess.ActiveRoleIDs) == 0 &&
			len(authorizedRoleIDs) > 0,
	}, nil
}

func (s *Service) authorizedRoleSummaries(
	ctx context.Context,
	userID pulid.ID,
	orgID pulid.ID,
) ([]services.RoleSummary, error) {
	roles, err := s.rbacRepo.GetAuthorizedRoles(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}
	summaries := make([]services.RoleSummary, 0, len(roles))
	for _, role := range roles {
		if role != nil {
			summaries = append(summaries, services.NewRoleSummary(role))
		}
	}
	return summaries, nil
}

func roleSummaryIDs(roles []services.RoleSummary) []pulid.ID {
	roleIDs := make([]pulid.ID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}
	return roleIDs
}

func activeRoleSummaries(
	activeRoleIDs []pulid.ID,
	authorizedRoles []services.RoleSummary,
) []services.RoleSummary {
	if len(activeRoleIDs) == 0 || len(authorizedRoles) == 0 {
		return []services.RoleSummary{}
	}

	byID := make(map[pulid.ID]services.RoleSummary, len(authorizedRoles))
	for _, role := range authorizedRoles {
		byID[role.ID] = role
	}

	activeRoles := make([]services.RoleSummary, 0, len(activeRoleIDs))
	for _, roleID := range activeRoleIDs {
		role, ok := byID[roleID]
		if ok {
			activeRoles = append(activeRoles, role)
		}
	}
	return activeRoles
}

func (s *Service) resolveRequestedOrganization(
	ctx context.Context,
	organizationSlug string,
	user *tenant.User,
) (*tenant.Organization, error) {
	if strings.TrimSpace(organizationSlug) == "" {
		return nil, nil //nolint:nilnil // nil result represents an optional absence in this API
	}

	org, err := s.or.GetByLoginSlug(ctx, organizationSlug)
	if err != nil {
		return nil, err
	}

	if err = s.ensureUserHasOrganizationAccess(ctx, user.ID, org.ID); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *Service) ensureUserHasOrganizationAccess(
	ctx context.Context,
	userID, organizationID pulid.ID,
) error {
	memberships, err := s.ur.GetOrganizations(ctx, userID)
	if err != nil {
		return err
	}

	for _, membership := range memberships {
		if membership.OrganizationID == organizationID {
			return nil
		}
	}

	return errortypes.NewAuthorizationError("You do not have access to this organization")
}

func (s *Service) enforcePasswordLoginPolicy(
	ctx context.Context,
	organizationSlug string,
	user *tenant.User,
	targetOrg *tenant.Organization,
) error {
	if strings.TrimSpace(organizationSlug) == "" && targetOrg == nil {
		targetOrg = &tenant.Organization{ID: user.CurrentOrganizationID}
	}

	if targetOrg == nil {
		return nil
	}

	if s.ssoRepo == nil {
		return nil
	}

	for _, p := range []tenant.SSOProvider{tenant.SSOProviderAzureAD, tenant.SSOProviderOkta} {
		cfg, err := s.ssoRepo.GetEnabledByOrganizationID(ctx, targetOrg.ID, p)
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				continue
			}
			return err
		}
		if cfg.EnforceSSO {
			return errSSORequired
		}
	}

	return nil
}

type oidcClaims struct {
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	UPN               string   `json:"upn"`
	Nonce             string   `json:"nonce"`
	TenantID          string   `json:"tid"`
	Subject           string   `json:"sub"`
	ACR               string   `json:"acr"`
	AMR               []string `json:"amr"`
}

func (c oidcClaims) EmailAddress() string {
	switch {
	case strings.TrimSpace(c.Email) != "":
		return strings.ToLower(strings.TrimSpace(c.Email))
	case strings.TrimSpace(c.PreferredUsername) != "":
		return strings.ToLower(strings.TrimSpace(c.PreferredUsername))
	default:
		return strings.ToLower(strings.TrimSpace(c.UPN))
	}
}

func assuranceFromOIDCClaims(claims oidcClaims) (int, int64) {
	for _, method := range claims.AMR {
		switch strings.ToLower(strings.TrimSpace(method)) {
		case "mfa", "otp", "totp", "webauthn", "hwk", "swk", "fido", "face", "fingerprint":
			return 2, timeutils.NowUnix()
		}
	}

	acr := strings.ToLower(strings.TrimSpace(claims.ACR))
	if strings.Contains(acr, "aal2") || strings.Contains(acr, "level2") ||
		strings.Contains(acr, "urn:mace:incommon:iap:silver") {
		return 2, timeutils.NowUnix()
	}

	return 1, 0
}

func validateReturnToURL(returnTo string, allowedOrigins []string) error {
	if strings.TrimSpace(returnTo) == "" {
		return errortypes.NewValidationError(
			"returnTo",
			errortypes.ErrRequired,
			"Return URL is required",
		)
	}

	parsed, err := url.Parse(returnTo)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errortypes.NewValidationError(
			"returnTo",
			errortypes.ErrInvalid,
			"Return URL must be an absolute URL",
		)
	}

	origin := parsed.Scheme + "://" + parsed.Host
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == origin {
			return nil
		}
	}

	return errortypes.NewValidationError(
		"returnTo",
		errortypes.ErrInvalid,
		"Return URL is not allowed for this environment",
	)
}

func validateAllowedDomain(emailAddress string, allowedDomains []string) error {
	if len(allowedDomains) == 0 {
		return nil
	}

	parts := strings.Split(emailAddress, "@")
	if len(parts) != 2 {
		return errortypes.NewAuthenticationError("SSO account email address is invalid")
	}

	for _, domain := range allowedDomains {
		if strings.EqualFold(strings.TrimSpace(domain), parts[1]) {
			return nil
		}
	}

	return errortypes.NewAuthenticationError(
		"SSO account email domain is not allowed for this organization",
	)
}

func providerDisplayName(p tenant.SSOProvider) string {
	switch p {
	case tenant.SSOProviderAzureAD:
		return "Microsoft"
	case tenant.SSOProviderOkta:
		return "Okta"
	default:
		return string(p)
	}
}

func microsoftTenantIDFromIssuer(issuerURL string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(issuerURL), "/"), "/")
	if len(parts) < 2 {
		return ""
	}

	return parts[len(parts)-2]
}

func randomURLToken(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(buf)
}

func mustDecryptSecret(enc *encryptionservice.Service, secret string) string {
	plaintext, err := enc.DecryptString(secret)
	if err != nil {
		return ""
	}

	return plaintext
}

func decryptSSOSecret(enc *encryptionservice.Service, cfg *tenant.SSOConfig) string {
	if cfg == nil {
		return ""
	}
	if enc == nil {
		return ""
	}

	if encryptionservice.IsEnvelope(cfg.OIDCClientSecret) {
		for _, resourceID := range ssoSecretAADResourceIDs(cfg.Provider) {
			plaintext, err := enc.DecryptStringWithAAD(
				cfg.OIDCClientSecret,
				encryptionservice.AAD{
					Purpose:        encryptionservice.PurposeIAMOIDCClientSecret,
					OrganizationID: cfg.OrganizationID,
					BusinessUnitID: cfg.BusinessUnitID,
					ResourceID:     resourceID,
				},
			)
			if err == nil {
				return plaintext
			}
		}
		return ""
	}

	return ""
}

func ssoSecretAADResourceIDs(provider tenant.SSOProvider) []string {
	resourceID := string(provider)
	normalized := strings.ToLower(strings.TrimSpace(resourceID))
	if normalized == "" || normalized == resourceID {
		return []string{resourceID}
	}

	return []string{resourceID, normalized}
}
