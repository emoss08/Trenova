package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/user"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type GetUserByIDOptions struct {
	OrgID        pulid.ID
	BuID         pulid.ID
	UserID       pulid.ID
	IncludeRoles bool
	IncludeOrgs  bool
}

type UserRepository interface {
	List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*user.User], error)
	FindByEmail(ctx context.Context, email string) (*user.User, error)
	UpdateLastLogin(ctx context.Context, userID pulid.ID) error
	GetByID(ctx context.Context, opts *GetUserByIDOptions) (*user.User, error)
}
