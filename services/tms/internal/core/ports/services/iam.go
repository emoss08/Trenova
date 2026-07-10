package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type IdentityProviderRequest struct {
	ID                pulid.ID                     `json:"id,omitempty"`
	Name              string                       `json:"name"`
	Slug              string                       `json:"slug"`
	Protocol          iam.IdentityProviderProtocol `json:"protocol"`
	Enabled           bool                         `json:"enabled"`
	EnforceSSO        bool                         `json:"enforceSso"`
	AutoProvision     bool                         `json:"autoProvision"`
	AllowFederatedMFA bool                         `json:"allowFederatedMfa"`
	AllowedDomains    []string                     `json:"allowedDomains"`
	AttributeMap      map[string]string            `json:"attributeMap"`
	OIDCIssuerURL     string                       `json:"oidcIssuerUrl"`
	OIDCClientID      string                       `json:"oidcClientId"`
	OIDCClientSecret  string                       `json:"oidcClientSecret,omitempty"`
	OIDCRedirectURL   string                       `json:"oidcRedirectUrl"`
	OIDCScopes        []string                     `json:"oidcScopes"`
	Version           int64                        `json:"version,omitempty"`
}

type SCIMTokenCreateRequest struct {
	Name      string `json:"name"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
}

type SCIMTokenCreateResponse struct {
	*iam.SCIMToken
	Token string `json:"token"`
}

type IAMService interface {
	ListIdentityProviders(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*iam.IdentityProvider, error)
	CreateIdentityProvider(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		req *IdentityProviderRequest,
	) (*iam.IdentityProvider, error)
	UpdateIdentityProvider(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		req *IdentityProviderRequest,
	) (*iam.IdentityProvider, error)
	DeleteIdentityProvider(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListSCIMDirectories(
		ctx context.Context,
		req *repositories.ListSCIMDirectoryRequest,
	) (*pagination.ListResult[*iam.SCIMDirectory], error)
	CreateSCIMDirectory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		entity *iam.SCIMDirectory,
	) (*iam.SCIMDirectory, error)
	UpdateSCIMDirectory(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		entity *iam.SCIMDirectory,
	) (*iam.SCIMDirectory, error)
	DeleteSCIMDirectory(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListSCIMTokens(ctx context.Context, orgID, directoryID pulid.ID) ([]*iam.SCIMToken, error)
	CreateSCIMToken(
		ctx context.Context,
		orgID pulid.ID,
		directoryID pulid.ID,
		req *SCIMTokenCreateRequest,
	) (*SCIMTokenCreateResponse, error)
	RevokeSCIMToken(ctx context.Context, orgID, tokenID pulid.ID) (*iam.SCIMToken, error)

	ListSCIMGroupRoleMappings(
		ctx context.Context,
		req *repositories.ListSCIMGroupRoleMappingsRequest,
	) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error)
	ListSCIMGroupRoleMappingsConnection(
		ctx context.Context,
		req *repositories.ListSCIMGroupRoleMappingConnectionRequest,
	) (*pagination.CursorListResult[*iam.SCIMGroupRoleMapping], error)
	CreateSCIMGroupRoleMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		directoryID pulid.ID,
		entity *iam.SCIMGroupRoleMapping,
	) (*iam.SCIMGroupRoleMapping, error)
	UpdateSCIMGroupRoleMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		entity *iam.SCIMGroupRoleMapping,
	) (*iam.SCIMGroupRoleMapping, error)
	DeleteSCIMGroupRoleMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) error

	ListProvisioningAuditRecords(
		ctx context.Context,
		orgID pulid.ID,
		directoryID pulid.ID,
		limit int,
	) ([]*iam.ProvisioningAuditRecord, error)

	ListAccessPolicies(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*iam.AccessPolicy, error)
	CreateAccessPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		entity *iam.AccessPolicy,
	) (*iam.AccessPolicy, error)
	UpdateAccessPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		entity *iam.AccessPolicy,
	) (*iam.AccessPolicy, error)
	DeleteAccessPolicy(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListAuthEvents(ctx context.Context, orgID pulid.ID, limit int) ([]*iam.AuthEvent, error)
	ListRiskDecisions(ctx context.Context, orgID pulid.ID, limit int) ([]*iam.RiskDecision, error)
	ListExternalIdentities(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*iam.ExternalIdentity, error)
	ListMFAAuthenticators(
		ctx context.Context,
		orgID pulid.ID,
		limit int,
	) ([]*iam.MFAAuthenticator, error)
}
