package roleassignmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.RoleAssignmentRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.RoleAssignmentRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.roleassignment"),
		repo: p.Repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRoleAssignmentsRequest,
) (*pagination.ListResult[*permission.UserRoleAssignment], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetRoleAssignmentByIDRequest,
) (*permission.UserRoleAssignment, error) {
	return s.repo.GetByID(ctx, req)
}
