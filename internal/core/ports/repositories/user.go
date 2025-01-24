package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
	GetByID(ctx context.Context, opts GetUserByIDOptions) (*user.User, error)
}
