package userservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/passwordresetservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/roleservice"
)

type RoleAssigner interface {
	AssignRole(ctx context.Context, req roleservice.AssignRoleRequest) error
}

type AccountInviter interface {
	SendInvite(ctx context.Context, req passwordresetservice.InviteRequest) error
}
