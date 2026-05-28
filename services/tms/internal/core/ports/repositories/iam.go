package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListIAMRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type IAMPolicyLookupRequest struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Resource       string
	Operation      permission.Operation
}

type ListSCIMDirectoryRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type GetSCIMDirectoryRequest struct {
	ID         pulid.ID              `json:"id"         form:"id"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo" form:"tenantInfo"`
}

type ListSCIMGroupRoleMappingsRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	DirectoryID pulid.ID                 `json:"directoryId" form:"directoryId"`
}

type IAMRepository interface {
	ListIdentityProviders(ctx context.Context, req ListIAMRequest) ([]*iam.IdentityProvider, error)
	GetIdentityProvider(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*iam.IdentityProvider, error)
	CreateIdentityProvider(
		ctx context.Context,
		entity *iam.IdentityProvider,
	) (*iam.IdentityProvider, error)
	UpdateIdentityProvider(
		ctx context.Context,
		entity *iam.IdentityProvider,
	) (*iam.IdentityProvider, error)
	DeleteIdentityProvider(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListSCIMDirectories(
		ctx context.Context,
		req *ListSCIMDirectoryRequest,
	) (*pagination.ListResult[*iam.SCIMDirectory], error)
	GetSCIMDirectory(
		ctx context.Context,
		req GetSCIMDirectoryRequest,
	) (*iam.SCIMDirectory, error)
	CreateSCIMDirectory(ctx context.Context, entity *iam.SCIMDirectory) (*iam.SCIMDirectory, error)
	UpdateSCIMDirectory(ctx context.Context, entity *iam.SCIMDirectory) (*iam.SCIMDirectory, error)
	DeleteSCIMDirectory(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListSCIMTokens(ctx context.Context, orgID, directoryID pulid.ID) ([]*iam.SCIMToken, error)
	CreateSCIMToken(ctx context.Context, entity *iam.SCIMToken) (*iam.SCIMToken, error)
	RevokeSCIMToken(ctx context.Context, orgID, tokenID pulid.ID) (*iam.SCIMToken, error)

	ListSCIMGroupRoleMappings(
		ctx context.Context,
		req *ListSCIMGroupRoleMappingsRequest,
	) (*pagination.ListResult[*iam.SCIMGroupRoleMapping], error)
	CreateSCIMGroupRoleMapping(
		ctx context.Context,
		entity *iam.SCIMGroupRoleMapping,
	) (*iam.SCIMGroupRoleMapping, error)
	UpdateSCIMGroupRoleMapping(
		ctx context.Context,
		entity *iam.SCIMGroupRoleMapping,
	) (*iam.SCIMGroupRoleMapping, error)
	DeleteSCIMGroupRoleMapping(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		sgrmID pulid.ID,
	) error

	ListProvisioningAuditRecords(
		ctx context.Context,
		orgID pulid.ID,
		directoryID pulid.ID,
		limit int,
	) ([]*iam.ProvisioningAuditRecord, error)

	ListAccessPolicies(ctx context.Context, req ListIAMRequest) ([]*iam.AccessPolicy, error)
	ListEnabledAccessPolicies(
		ctx context.Context,
		req IAMPolicyLookupRequest,
	) ([]*iam.AccessPolicy, error)
	CreateAccessPolicy(ctx context.Context, entity *iam.AccessPolicy) (*iam.AccessPolicy, error)
	UpdateAccessPolicy(ctx context.Context, entity *iam.AccessPolicy) (*iam.AccessPolicy, error)
	DeleteAccessPolicy(ctx context.Context, tenantInfo pagination.TenantInfo, id pulid.ID) error

	ListAuthEvents(ctx context.Context, orgID pulid.ID, limit int) ([]*iam.AuthEvent, error)
	ListRiskDecisions(ctx context.Context, orgID pulid.ID, limit int) ([]*iam.RiskDecision, error)
	ListExternalIdentities(ctx context.Context, req ListIAMRequest) ([]*iam.ExternalIdentity, error)
	ListMFAAuthenticators(
		ctx context.Context,
		orgID pulid.ID,
		limit int,
	) ([]*iam.MFAAuthenticator, error)
}
