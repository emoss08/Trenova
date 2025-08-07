/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetUserByIDOptions struct {
	OrgID        pulid.ID
	BuID         pulid.ID
	UserID       pulid.ID
	IncludeRoles bool
	IncludeOrgs  bool
}

type ListUserRequest struct {
	Filter       *ports.LimitOffsetQueryOptions
	IncludeRoles bool
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

type GetUsersByIDsOptions struct {
	OrgID   pulid.ID
	BuID    pulid.ID
	UserIDs []pulid.ID
}

type UserRepository interface {
	List(
		ctx context.Context,
		req ListUserRequest,
	) (*ports.ListResult[*user.User], error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	GetNameByID(ctx context.Context, userID pulid.ID) (string, error)
	GetByID(ctx context.Context, opts GetUserByIDOptions) (*user.User, error)
	GetByIDs(ctx context.Context, opts GetUsersByIDsOptions) ([]*user.User, error)
	GetSystemUser(ctx context.Context) (*user.User, error)
	UpdateLastLogin(ctx context.Context, userID pulid.ID) error
	Create(ctx context.Context, u *user.User) (*user.User, error)
	Update(ctx context.Context, u *user.User) (*user.User, error)
	SwitchOrganization(ctx context.Context, userID, newOrgID pulid.ID) (*user.User, error)
	ChangePassword(ctx context.Context, req *ChangePasswordRequest) (*user.User, error)
}
