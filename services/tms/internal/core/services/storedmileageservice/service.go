package storedmileageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/storedmileage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.StoredMileageRepository
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.StoredMileageRepository
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.stored-mileage"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListStoredMileageRequest,
) (*pagination.ListResult[*storedmileage.StoredMileage], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetStoredMileageByIDRequest,
) (*storedmileage.StoredMileage, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Deactivate(
	ctx context.Context,
	req repositories.DeleteStoredMileageRequest,
	userID pulid.ID,
) error {
	existing, err := s.repo.GetByID(ctx, repositories.GetStoredMileageByIDRequest(req))
	if err != nil {
		return err
	}
	if err = s.repo.Deactivate(ctx, req); err != nil {
		return err
	}
	current := *existing
	current.Status = storedmileage.StatusInactive
	s.logAudit(&current, existing, userID)
	return nil
}

func (s *Service) logAudit(
	current *storedmileage.StoredMileage,
	previous *storedmileage.StoredMileage,
	userID pulid.ID,
) {
	if current == nil {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceStoredMileage,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	opts := []services.LogOption{auditservice.WithComment("Stored mileage deactivated")}
	if previous != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log stored mileage audit action", zap.Error(err))
	}
}
