package roleassignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
) (*pagination.ListResult[*permission.UserRoleAssignment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req permissionports.GetRoleAssignmentByIDRequest,
) (*permission.UserRoleAssignment, error) {
	return s.repo.GetByID(ctx, req)
}
