package roleassignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/rbac"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   permissionports.RoleAssignmentRepository
}

type Service struct {
	l    *zap.Logger
	repo permissionports.RoleAssignmentRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.roleassignment"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *permissionports.ListRoleAssignmentsRequest,
) (*pagination.ListResult[*rbac.UserRoleAssignment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req permissionports.GetRoleAssignmentByIDRequest,
) (*rbac.UserRoleAssignment, error) {
	return s.repo.GetByID(ctx, req)
}
