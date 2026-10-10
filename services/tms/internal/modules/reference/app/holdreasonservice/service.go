package holdreasonservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/reference/ports/holdreasonports"
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
	Repo         holdreasonports.HoldReasonRepository
	Validator    *Validator
	AuditService auditports.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         holdreasonports.HoldReasonRepository
	validator    *Validator
	auditService auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.hold-reason"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *holdreasonports.ListHoldReasonRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *holdreasonports.ListHoldReasonConnectionRequest,
) (*pagination.CursorListResult[*holdreason.HoldReason], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req holdreasonports.GetHoldReasonByIDRequest,
) (*holdreason.HoldReason, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *holdreasonports.HoldReasonSelectOptionsRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *holdreason.HoldReason,
	userID pulid.ID,
) (*holdreason.HoldReason, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create hold reason", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceHoldReason,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Hold reason created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *holdreason.HoldReason,
	userID pulid.ID,
) (*holdreason.HoldReason, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, holdreasonports.GetHoldReasonByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original hold reason", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update hold reason", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceHoldReason,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Hold reason updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
