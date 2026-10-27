package tenantports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetUserByIDRequest struct {
	TenantInfo         pagination.TenantInfo
	LookupUserID       pulid.ID
	IncludeMemberships bool `json:"includeMemberships"`
}

type UserOrganizationResult struct {
	OrganizationID   pulid.ID
	OrganizationName string
	BusinessUnitID   pulid.ID
	IsDefault        bool
	JoinedAt         int64
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

type ListUserConnectionRequest struct {
	Filter      *pagination.QueryOptions `json:"filter"`
	Cursor      pagination.CursorInfo    `json:"-"`
	UserColumns []string                 `json:"-"`
}

type BulkUpdateUserStatusRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	UserIDs    []pulid.ID            `json:"userIds"`
	Status     domaintypes.Status    `json:"status"`
}

type GetTenantMemberRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
	Now        int64
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

type UpdateUserColumnsRequest struct {
	Entity  *tenant.User
	Columns []string
}

type CreateUserRequest struct {
	User       *tenant.User
	Membership *tenant.OrganizationMembership
}

type UserIdentityConflictsRequest struct {
	EmailAddress  string
	Username      string
	ExcludeUserID pulid.ID
}

type UserIdentityConflicts struct {
	EmailAddress bool
	Username     bool
}

type UserRepository interface {
	Create(ctx context.Context, req CreateUserRequest) (*tenant.User, error)
	FindIdentityConflicts(
		ctx context.Context,
		req UserIdentityConflictsRequest,
	) (UserIdentityConflicts, error)
	List(ctx context.Context, req *ListUsersRequest) (*pagination.ListResult[*tenant.User], error)
	ListConnection(
		ctx context.Context,
		req *ListUserConnectionRequest,
	) (*pagination.CursorListResult[*tenant.User], error)
	GetByID(ctx context.Context, req GetUserByIDRequest) (*tenant.User, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*tenant.User], error)
	FindByEmail(ctx context.Context, emailAddress string) (*tenant.User, error)
	FindByIDForLogin(ctx context.Context, userID pulid.ID) (*tenant.User, error)
	UpdateLastLoginAt(ctx context.Context, userID pulid.ID) error
	GetOrganizations(
		ctx context.Context,
		userID pulid.ID,
	) ([]*tenant.OrganizationMembership, error)
	ListOrganizationMemberships(
		ctx context.Context,
		userID, businessUnitID pulid.ID,
	) ([]*tenant.OrganizationMembership, error)
	ReplaceOrganizationMemberships(
		ctx context.Context,
		req ReplaceOrganizationMembershipsRequest,
	) ([]*tenant.OrganizationMembership, error)
	UpdateCurrentOrganization(ctx context.Context, userID, orgID, buID pulid.ID) error
	GetUserOrganizationSummaries(ctx context.Context, userID pulid.ID) ([]OrgSummary, error)
	UpdateColumns(ctx context.Context, req UpdateUserColumnsRequest) (*tenant.User, error)
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
	GetTenantMember(ctx context.Context, req GetTenantMemberRequest) (*tenant.User, error)
}
