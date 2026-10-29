package fleetcodeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         fleetcodeports.FleetCodeRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Transformer  accessorialchargeports.DataTransformer
}

type Service struct {
	l            *zap.Logger
	repo         fleetcodeports.FleetCodeRepository
	validator    *Validator
	auditService auditports.AuditService
	transformer  accessorialchargeports.DataTransformer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.fleetcode"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		transformer:  p.Transformer,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *fleetcodeports.ListFleetCodesRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *fleetcodeports.ListFleetCodeConnectionRequest,
) (*pagination.CursorListResult[*fleetcode.FleetCode], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req fleetcodeports.GetFleetCodeByIDRequest,
) (*fleetcode.FleetCode, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req fleetcodeports.GetFleetCodesByIDsRequest,
) ([]*fleetcode.FleetCode, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*fleetcode.FleetCode], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *fleetcode.FleetCode,
	userID pulid.ID,
) (*fleetcode.FleetCode, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformFleetCode(ctx, entity); err != nil {
		log.Error("failed to transform fleet code", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create fleet code", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceFleetCode,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fleet code created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *fleetcode.FleetCode,
	userID pulid.ID,
) (*fleetcode.FleetCode, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformFleetCode(ctx, entity); err != nil {
		log.Error("failed to transform fleet code", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, fleetcodeports.GetFleetCodeByIDRequest{
		ID: entity.GetID(),
		TenantInfo: &pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original fleet code", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update fleet code", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceFleetCode,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Fleet code updated"),
		auditservice.WithDiff(original, updatedEntity)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
