package dedicatedlane

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DedicatedLaneRepository
	AuditService services.AuditService
}

type Service struct {
	l    *zap.Logger
	repo repositories.DedicatedLaneRepository
	as   services.AuditService
}

func NewService(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.dedicatedlane"),
		repo: p.Repo,
		as:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneRequest,
) (*pagination.ListResult[*dedicatedlane.DedicatedLane], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *repositories.GetDedicatedLaneByIDRequest,
) (*dedicatedlane.DedicatedLane, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) FindByShipment(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.DedicatedLane, error) {
	return s.repo.FindByShipment(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *dedicatedlane.DedicatedLane,
	userID pulid.ID,
) (*dedicatedlane.DedicatedLane, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDedicatedLane,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Dedicated lane created"),
	)
	if err != nil {
		log.Error("failed to log dedicated lane creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *dedicatedlane.DedicatedLane,
	userID pulid.ID,
) (*dedicatedlane.DedicatedLane, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByID(ctx, &repositories.GetDedicatedLaneByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update dedicated lane", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDedicatedLane,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Dedicated lane updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log dedicated lane update", zap.Error(err))
	}

	return updatedEntity, nil
}
