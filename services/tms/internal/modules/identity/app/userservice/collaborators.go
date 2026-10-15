package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
)

type RoleAssigner interface {
	AssignRole(ctx context.Context, req roleservice.AssignRoleRequest) error
	RevokeOrganizationRoles(
		ctx context.Context,
		req roleservice.RevokeOrganizationRolesRequest,
	) error
}

type AccountInviter interface {
	SendInvite(ctx context.Context, req passwordresetservice.InviteRequest) error
}
