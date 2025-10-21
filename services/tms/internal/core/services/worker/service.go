package worker

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/workervalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger             *zap.Logger
	Repo               repositories.WorkerRepository
	AuditService       services.AuditService
	Validator          *workervalidator.Validator
	WorkerPTOValidator *workervalidator.WorkerPTOValidator
}

type Service struct {
	l    *zap.Logger
	repo repositories.WorkerRepository
	as   services.AuditService
	v    *workervalidator.Validator
	wpv  *workervalidator.WorkerPTOValidator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.worker"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
		wpv:  p.WorkerPTOValidator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListWorkerRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *worker.Worker,
	userID pulid.ID,
) (*worker.Worker, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorker,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Worker created"),
	)
	if err != nil {
		log.Error("failed to log worker creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.Worker,
	userID pulid.ID,
) (*worker.Worker, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetWorkerByIDRequest{
		WorkerID: entity.ID,
		OrgID:    entity.OrganizationID,
		BuID:     entity.BusinessUnitID,
		FilterOptions: repositories.WorkerFilterOptions{
			IncludeProfile: true,
			IncludePTO:     true,
		},
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update worker", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorker,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Worker updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log worker update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) ListUpcomingPTO(
	ctx context.Context,
	req *repositories.ListUpcomingWorkerPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	return s.repo.ListUpcomingPTO(ctx, req)
}

func (s *Service) ApprovePTO(
	ctx context.Context,
	req *repositories.ApprovePTORequest,
) error {
	return s.repo.ApprovePTO(ctx, req)
}

func (s *Service) RejectPTO(
	ctx context.Context,
	req *repositories.RejectPTORequest,
) error {
	return s.repo.RejectPTO(ctx, req)
}

func (s *Service) ListWorkerPTO(
	ctx context.Context,
	req *repositories.ListWorkerPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	return s.repo.ListWorkerPTO(ctx, req)
}

func (s *Service) GetPTOChartData(
	ctx context.Context,
	req *repositories.PTOChartDataRequest,
) ([]*repositories.PTOChartDataPoint, error) {
	return s.repo.GetPTOChartData(ctx, req)
}

func (s *Service) GetPTOCalendarData(
	ctx context.Context,
	req *repositories.PTOCalendarDataRequest,
) ([]*repositories.PTOCalendarEvent, error) {
	return s.repo.GetPTOCalendarData(ctx, req)
}

func (s *Service) CreateWorkerPTO(
	ctx context.Context,
	pto *worker.WorkerPTO,
	userID pulid.ID,
) (*worker.WorkerPTO, error) {
	log := s.l.With(
		zap.String("operation", "CreateWorkerPTO"),
		zap.Any("pto", pto),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.wpv.Validate(ctx, valCtx, pto); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.CreateWorkerPTO(ctx, pto)
	if err != nil {
		log.Error("failed to create worker PTO", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceWorkerPTO,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Worker PTO created"),
	)
	if err != nil {
		log.Error("failed to log worker PTO creation", zap.Error(err))
	}

	return createdEntity, nil
}
