package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetUserByIDRequest struct {
	OrgID        pulid.ID
	BuID         pulid.ID
	UserID       pulid.ID
	IncludeRoles bool
	IncludeOrgs  bool
}

type UserSelectOptionsRequest struct {
	*pagination.SelectQueryOptions
}

type UserSelectOptionResponse struct {
	ID            pulid.ID `json:"id"            form:"id"            bun:"id"`
	Name          string   `json:"name"          form:"name"          bun:"name"`
	Username      string   `json:"username"      form:"username"      bun:"username"`
	ProfilePicURL string   `json:"profilePicUrl" form:"profilePicUrl" bun:"profile_pic_url"`
	EmailAddress  string   `json:"emailAddress"  form:"emailAddress"  bun:"email_address"`
}

type ChangePasswordRequest struct {
	OrgID           pulid.ID `json:"orgId"`
	BuID            pulid.ID `json:"buId"`
	UserID          pulid.ID `json:"userId"`
	CurrentPassword string   `json:"currentPassword"`
	NewPassword     string   `json:"newPassword"`
	ConfirmPassword string   `json:"confirmPassword"`
	HashedPassword  string   `json:"-"`
}

type GetUsersByIDsRequest struct {
	OrgID   pulid.ID
	BuID    pulid.ID
	UserIDs []pulid.ID
}

type ListUserRequest struct {
	Filter       *pagination.QueryOptions
	IncludeRoles bool
}

type UpdateMeRequest struct {
	UserID       pulid.ID               `json:"userId"`
	OrgID        pulid.ID               `json:"orgId"`
	BuID         pulid.ID               `json:"buId"`
	Name         string                 `json:"name"         form:"name"`
	Username     string                 `json:"username"     form:"username"`
	EmailAddress string                 `json:"emailAddress" form:"emailAddress"`
	Timezone     string                 `json:"timezone"     form:"timezone"`
	TimeFormat   domaintypes.TimeFormat `json:"timeFormat"   form:"timeFormat"`
}

type UserRepository interface {
	GetOption(ctx context.Context, req GetUserByIDRequest) (*tenant.User, error)
	SelectOptions(
		ctx context.Context,
		req UserSelectOptionsRequest,
	) ([]*UserSelectOptionResponse, error)
	List(ctx context.Context, req *ListUserRequest) (*pagination.ListResult[*tenant.User], error)
	FindByEmail(ctx context.Context, email string) (*tenant.User, error)
	GetNameByID(ctx context.Context, userID pulid.ID) (string, error)
	GetByID(ctx context.Context, opts GetUserByIDRequest) (*tenant.User, error)
	GetByIDs(ctx context.Context, opts GetUsersByIDsRequest) ([]*tenant.User, error)
	GetSystemUser(ctx context.Context) (*tenant.User, error)
	UpdateLastLogin(ctx context.Context, userID pulid.ID) error
	Update(ctx context.Context, u *tenant.User) (*tenant.User, error)
	UpdateMe(ctx context.Context, req *UpdateMeRequest) (*tenant.User, error)
	Create(ctx context.Context, u *tenant.User) (*tenant.User, error)
	SwitchOrganization(ctx context.Context, userID, newOrgID pulid.ID) (*tenant.User, error)
	ChangePassword(ctx context.Context, req *ChangePasswordRequest) (*tenant.User, error)
}
