package iamservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/slugutil"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	identityProviderSlugMaxLength = 80
	scimTokenPrefix               = "scim_"
)

type Params struct {
	fx.In

	Repo       repositories.IAMRepository
	Encryption *encryptionservice.Service
	Validator  *Validator
	Logger     *zap.Logger
}

type service struct {
	repo      repositories.IAMRepository
	enc       *encryptionservice.Service
	validator *Validator
	l         *zap.Logger
}

func New(p Params) services.IAMService {
	return &service{
		repo:      p.Repo,
		enc:       p.Encryption,
		validator: p.Validator,
		l:         p.Logger.Named("service.iam"),
	}
}

func (s *service) ListIdentityProviders(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*iam.IdentityProvider, error) {
	return s.repo.ListIdentityProviders(ctx, repositories.ListIAMRequest{TenantInfo: tenantInfo})
}

func (s *service) CreateIdentityProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *services.IdentityProviderRequest,
) (*iam.IdentityProvider, error) {
	entity, err := s.identityProviderFromRequest(tenantInfo, req, nil)
	if err != nil {
		return nil, err
	}
	return s.repo.CreateIdentityProvider(ctx, entity)
}

func (s *service) UpdateIdentityProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	req *services.IdentityProviderRequest,
) (*iam.IdentityProvider, error) {
	existing, err := s.repo.GetIdentityProvider(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}

	entity, err := s.identityProviderFromRequest(tenantInfo, req, existing)
	if err != nil {
		return nil, err
	}
	entity.ID = id
	entity.Version = existing.Version
	if strings.TrimSpace(req.OIDCClientSecret) == "" {
		entity.OIDCClientSecret = existing.OIDCClientSecret
	}

	return s.repo.UpdateIdentityProvider(ctx, entity)
}

func (s *service) DeleteIdentityProvider(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.repo.DeleteIdentityProvider(ctx, tenantInfo, id)
}

func (s *service) identityProviderFromRequest(
	tenantInfo pagination.TenantInfo,
	req *services.IdentityProviderRequest,
	existing *iam.IdentityProvider,
) (*iam.IdentityProvider, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"identityProvider",
			errortypes.ErrRequired,
			"Identity provider is required",
		)
	}

	protocol := req.Protocol
	if protocol == "" {
		protocol = iam.IdentityProviderProtocolOIDC
	}
	if protocol != iam.IdentityProviderProtocolOIDC {
		return nil, errortypes.NewValidationError(
			"protocol",
			errortypes.ErrInvalid,
			"SAML providers cannot be managed until SAML sign-in is available",
		)
	}

	slug := normalizeProviderSlug(req.Slug, req.Name)
	secret, err := s.resolveOIDCSecret(tenantInfo, slug, req.OIDCClientSecret, existing)
	if err != nil {
		return nil, err
	}

	entity := &iam.IdentityProvider{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Name:              strings.TrimSpace(req.Name),
		Slug:              slug,
		Protocol:          protocol,
		Enabled:           req.Enabled,
		EnforceSSO:        req.EnforceSSO,
		AutoProvision:     req.AutoProvision,
		AllowFederatedMFA: req.AllowFederatedMFA,
		AllowedDomains:    normalizedStrings(req.AllowedDomains),
		AttributeMap:      req.AttributeMap,
		OIDCIssuerURL:     strings.TrimSpace(req.OIDCIssuerURL),
		OIDCClientID:      strings.TrimSpace(req.OIDCClientID),
		OIDCClientSecret:  secret,
		OIDCRedirectURL:   strings.TrimSpace(req.OIDCRedirectURL),
		OIDCScopes:        normalizedScopes(req.OIDCScopes),
	}
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	validateOIDCConfig(entity, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return entity, nil
}

func (s *service) resolveOIDCSecret(
	tenantInfo pagination.TenantInfo,
	slug string,
	rawSecret string,
	existing *iam.IdentityProvider,
) (string, error) {
	trimmed := strings.TrimSpace(rawSecret)
	if trimmed == "" && existing != nil {
		return existing.OIDCClientSecret, nil
	}
	if trimmed == "" {
		return "", nil
	}
	encrypted, err := s.enc.EncryptStringWithAAD(trimmed, encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeIAMOIDCClientSecret,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     slug,
	})
	if err != nil {
		return "", errortypes.NewBusinessError("Failed to encrypt identity provider secret").
			WithInternal(err)
	}
	return encrypted, nil
}

func validateOIDCConfig(entity *iam.IdentityProvider, multiErr *errortypes.MultiError) {
	if entity.Protocol != iam.IdentityProviderProtocolOIDC {
		return
	}
	if entity.OIDCIssuerURL == "" {
		multiErr.Add("oidcIssuerUrl", errortypes.ErrRequired, "OIDC issuer URL is required")
	}
	if entity.OIDCClientID == "" {
		multiErr.Add("oidcClientId", errortypes.ErrRequired, "OIDC client ID is required")
	}
	if entity.OIDCClientSecret == "" {
		multiErr.Add("oidcClientSecret", errortypes.ErrRequired, "OIDC client secret is required")
	}
	if entity.OIDCRedirectURL == "" {
		multiErr.Add("oidcRedirectUrl", errortypes.ErrRequired, "OIDC redirect URL is required")
	}
}

func normalizeProviderSlug(rawSlug, name string) string {
	slug := slugutil.NormalizeLoginSlug(rawSlug)
	if strings.TrimSpace(rawSlug) == "" {
		slug = slugutil.NormalizeLoginSlug(name)
	}
	if len(slug) > identityProviderSlugMaxLength {
		slug = strings.Trim(slug[:identityProviderSlugMaxLength], "-")
	}
	if slug == "" {
		return "provider"
	}
	return slug
}

func normalizedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func normalizedScopes(values []string) []string {
	scopes := normalizedStrings(values)
	if len(scopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return scopes
}

func (s *service) ListSCIMDirectories(
	ctx context.Context,
	req *repositories.ListSCIMDirectoryRequest,
) (*pagination.ListResult[*iam.SCIMDirectory], error) {
	return s.repo.ListSCIMDirectories(ctx, req)
}

func (s *service) CreateSCIMDirectory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.TenantSlug = strings.TrimSpace(entity.TenantSlug)
	return s.repo.CreateSCIMDirectory(ctx, entity)
}

func (s *service) UpdateSCIMDirectory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	entity *iam.SCIMDirectory,
) (*iam.SCIMDirectory, error) {
	entity.ID = id
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.TenantSlug = strings.TrimSpace(entity.TenantSlug)
	return s.repo.UpdateSCIMDirectory(ctx, entity)
}

func (s *service) DeleteSCIMDirectory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.repo.DeleteSCIMDirectory(ctx, tenantInfo, id)
}

func (s *service) ListSCIMTokens(
	ctx context.Context,
	orgID, directoryID pulid.ID,
) ([]*iam.SCIMToken, error) {
	return s.repo.ListSCIMTokens(ctx, orgID, directoryID)
}

func (s *service) CreateSCIMToken(
	ctx context.Context,
	orgID pulid.ID,
	directoryID pulid.ID,
	req *services.SCIMTokenCreateRequest,
) (*services.SCIMTokenCreateResponse, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"token",
			errortypes.ErrRequired,
			"SCIM token request is required",
		)
	}

	generated, err := generateSCIMToken()
	if err != nil {
		return nil, errortypes.NewBusinessError("Failed to generate SCIM token").WithInternal(err)
	}

	entity := &iam.SCIMToken{
		OrganizationID: orgID,
		DirectoryID:    directoryID,
		Name:           strings.TrimSpace(req.Name),
		Prefix:         generated.prefix,
		TokenHash:      generated.hash,
		Status:         iam.SCIMTokenStatusActive,
		ExpiresAt:      req.ExpiresAt,
	}
	if entity.Name == "" {
		return nil, errortypes.NewValidationError(
			"name",
			errortypes.ErrRequired,
			"Name is required",
		)
	}

	saved, err := s.repo.CreateSCIMToken(ctx, entity)
	if err != nil {
		return nil, err
	}
	return &services.SCIMTokenCreateResponse{
		SCIMToken: saved,
		Token:     generated.token,
	}, nil
}

func (s *service) RevokeSCIMToken(
	ctx context.Context,
	orgID, tokenID pulid.ID,
) (*iam.SCIMToken, error) {
	return s.repo.RevokeSCIMToken(ctx, orgID, tokenID)
}

type generatedSCIMToken struct {
	prefix string
	token  string
	hash   string
}

func generateSCIMToken() (*generatedSCIMToken, error) {
	prefixBytes := make([]byte, 9)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(prefixBytes); err != nil {
		return nil, err
	}
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, err
	}

	prefix := scimTokenPrefix + base64.RawURLEncoding.EncodeToString(prefixBytes)
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)
	token := prefix + "." + secret
	sum := sha256.Sum256([]byte(token))
	return &generatedSCIMToken{
		prefix: prefix,
		token:  token,
		hash:   base64.RawURLEncoding.EncodeToString(sum[:]),
	}, nil
}

func (s *service) ListSCIMGroupRoleMappings(
	ctx context.Context,
	req *repositories.ListSCIMGroupRoleMappingsRequest,
) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error) {
	return s.repo.ListSCIMGroupRoleMappings(ctx, req)
}

func (s *service) CreateSCIMGroupRoleMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	directoryID pulid.ID,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.DirectoryID = directoryID
	return s.repo.CreateSCIMGroupRoleMapping(ctx, entity)
}

func (s *service) UpdateSCIMGroupRoleMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	entity *iam.SCIMGroupRoleMapping,
) (*iam.SCIMGroupRoleMapping, error) {
	entity.ID = id
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	return s.repo.UpdateSCIMGroupRoleMapping(ctx, entity)
}

func (s *service) DeleteSCIMGroupRoleMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.repo.DeleteSCIMGroupRoleMapping(ctx, tenantInfo, id)
}

func (s *service) ListProvisioningAuditRecords(
	ctx context.Context,
	orgID pulid.ID,
	directoryID pulid.ID,
	limit int,
) ([]*iam.ProvisioningAuditRecord, error) {
	return s.repo.ListProvisioningAuditRecords(ctx, orgID, directoryID, limit)
}

func (s *service) ListAccessPolicies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*iam.AccessPolicy, error) {
	return s.repo.ListAccessPolicies(ctx, repositories.ListIAMRequest{TenantInfo: tenantInfo})
}

func (s *service) CreateAccessPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	prepareAccessPolicy(entity)
	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.CreateAccessPolicy(ctx, entity)
}

func (s *service) UpdateAccessPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	entity *iam.AccessPolicy,
) (*iam.AccessPolicy, error) {
	entity.ID = id
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	prepareAccessPolicy(entity)
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}
	return s.repo.UpdateAccessPolicy(ctx, entity)
}

func prepareAccessPolicy(entity *iam.AccessPolicy) {
	entity.Name = strings.TrimSpace(entity.Name)
	entity.Resource = strings.TrimSpace(entity.Resource)
	entity.Operation = strings.TrimSpace(entity.Operation)
	entity.Conditions = normalizeAccessPolicyConditions(entity.Conditions)
}

func (s *service) DeleteAccessPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.repo.DeleteAccessPolicy(ctx, tenantInfo, id)
}

func (s *service) ListAuthEvents(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.AuthEvent, error) {
	return s.repo.ListAuthEvents(ctx, orgID, limit)
}

func (s *service) ListRiskDecisions(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.RiskDecision, error) {
	return s.repo.ListRiskDecisions(ctx, orgID, limit)
}

func (s *service) ListExternalIdentities(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*iam.ExternalIdentity, error) {
	return s.repo.ListExternalIdentities(ctx, repositories.ListIAMRequest{TenantInfo: tenantInfo})
}

func (s *service) ListMFAAuthenticators(
	ctx context.Context,
	orgID pulid.ID,
	limit int,
) ([]*iam.MFAAuthenticator, error) {
	return s.repo.ListMFAAuthenticators(ctx, orgID, limit)
}
