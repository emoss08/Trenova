package ratezoneservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratezone"
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
	Repo         repositories.RateZoneRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RateZoneRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.rate-zone"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateZoneRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRateZoneConnectionRequest,
) (*pagination.CursorListResult[*ratezone.RateZone], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratezone.RateZone], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateZoneByIDRequest,
) (*ratezone.RateZone, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *ratezone.RateZone,
	userID pulid.ID,
) (*ratezone.RateZone, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userId", userID.String()),
	)

	entity.StampMembers()

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create rate zone", zap.Error(err))
		return nil, err
	}

	s.audit(log, created, nil, permission.OpCreate, userID, "Rate zone created")

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *ratezone.RateZone,
	userID pulid.ID,
) (*ratezone.RateZone, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userId", userID.String()),
	)

	entity.StampMembers()

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetRateZoneByIDRequest{
		RateZoneID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeMembers: true,
	})
	if err != nil {
		log.Error("failed to load rate zone for update", zap.Error(err))
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update rate zone", zap.Error(err))
		return nil, err
	}

	s.audit(log, updated, original, permission.OpUpdate, userID, "Rate zone updated")

	return updated, nil
}

// Delete removes a zone. The database refuses if a rate rule still points at
// it, which is the right answer: silently deleting a zone would turn every lane
// written against it into a rate that can never match again.
func (s *Service) Delete(
	ctx context.Context,
	req *repositories.GetRateZoneByIDRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("rateZoneId", req.RateZoneID.String()),
	)

	existing, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to load rate zone for delete", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete rate zone", zap.Error(err))
		return err
	}

	s.audit(log, existing, nil, permission.OpDelete, userID, "Rate zone deleted")

	return nil
}

func (s *Service) audit(
	log *zap.Logger,
	entity *ratezone.RateZone,
	original *ratezone.RateZone,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceRateZone,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	if operation == permission.OpDelete {
		params.PreviousState = jsonutils.MustToJSON(entity)
	} else {
		params.CurrentState = jsonutils.MustToJSON(entity)
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}

	if original != nil {
		params.PreviousState = jsonutils.MustToJSON(original)
		opts = append(opts, auditservice.WithDiff(original, entity))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}
