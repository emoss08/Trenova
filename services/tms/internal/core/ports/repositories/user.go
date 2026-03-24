package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetUserByIDRequest struct {
	TenantInfo         pagination.TenantInfo
	IncludeMemberships bool `json:"includeMemberships"`
}

type UserOrganizationResult struct {
	OrganizationID   pulid.ID `bun:"organization_id"`
	OrganizationName string   `bun:"organization_name"`
	BusinessUnitID   pulid.ID `bun:"business_unit_id"`
	IsDefault        bool     `bun:"is_default"`
	JoinedAt         int64    `bun:"joined_at"`
}

type UserOrganizationResponse struct {
	ID        pulid.ID `json:"id"`
	Name      string   `json:"name"`
	City      string   `json:"city"`
	State     string   `json:"state"`
	LogoURL   string   `json:"logoUrl"`
	IsDefault bool     `json:"isDefault"`
	IsCurrent bool     `json:"isCurrent"`
}

type OrgSummary struct {
	ID   pulid.ID `json:"id"`
	Name string   `json:"name"`
}

type SwitchOrganizationRequest struct {
	SessionID      pulid.ID
	OrganizationID pulid.ID
}

type ListUsersRequest struct {
	Filter             *pagination.QueryOptions
	IncludeMemberships bool `json:"includeMemberships"`
}

type BulkUpdateUserStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	UserIDs    []pulid.ID            `json:"userIds"`
	Status     domaintypes.Status    `json:"status"`
}

type GetUsersByIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	UserIDs    []pulid.ID            `json:"userIds"`
}

type ReplaceOrganizationMembershipsRequest struct {
	ActorID         pulid.ID
	UserID          pulid.ID
	BusinessUnitID  pulid.ID
	OrganizationIDs []pulid.ID
}

type UpdateUserPasswordRequest struct {
	UserID             pulid.ID
	OrganizationID     pulid.ID
	BusinessUnitID     pulid.ID
	Password           string
	MustChangePassword bool
}

type UserRepository interface {
	List(ctx context.Context, req *ListUsersRequest) (*pagination.ListResult[*tenant.User], error)
	GetByID(ctx context.Context, req GetUserByIDRequest) (*tenant.User, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*tenant.User], error)
	FindByEmail(ctx context.Context, emailAddress string) (*tenant.User, error)
	UpdateLastLoginAt(ctx context.Context, userID pulid.ID) error
	GetOrganizations(
		ctx context.Context,
		userID pulid.ID,
	) ([]*tenant.OrganizationMembership, error)
	GetOrganizationsByBusinessUnit(
		ctx context.Context,
		businessUnitID pulid.ID,
	) ([]*tenant.Organization, error)
	GetOrganizationByID(
		ctx context.Context,
		organizationID pulid.ID,
	) (*tenant.Organization, error)
	ListOrganizationMemberships(
		ctx context.Context,
		userID, businessUnitID pulid.ID,
	) ([]*tenant.OrganizationMembership, error)
	ReplaceOrganizationMemberships(
		ctx context.Context,
		req ReplaceOrganizationMembershipsRequest,
	) ([]*tenant.OrganizationMembership, error)
	UpdateCurrentOrganization(ctx context.Context, userID, orgID, buID pulid.ID) error
	IsPlatformAdmin(ctx context.Context, userID pulid.ID) (bool, error)
	GetUserOrganizationSummaries(ctx context.Context, userID pulid.ID) ([]OrgSummary, error)
	Update(ctx context.Context, entity *tenant.User) (*tenant.User, error)
	UpdatePassword(ctx context.Context, req UpdateUserPasswordRequest) error
	BulkUpdateStatus(
		ctx context.Context,
		req *BulkUpdateUserStatusRequest,
	) ([]*tenant.User, error)
	GetByIDs(
		ctx context.Context,
		req GetUsersByIDsRequest,
	) ([]*tenant.User, error)
	GetSystemUser(ctx context.Context, columns ...string) (*tenant.User, error)
}
