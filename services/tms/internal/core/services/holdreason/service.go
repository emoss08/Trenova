package holdreason

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/holdreasonvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.HoldReasonRepository
	AuditService services.AuditService
	Validator    *holdreasonvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.HoldReasonRepository
	as   services.AuditService
	v    *holdreasonvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.hold_reason"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListHoldReasonRequest,
) (*pagination.ListResult[*holdreason.HoldReason], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetHoldReasonByIDRequest,
) (*holdreason.HoldReason, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	hr *holdreason.HoldReason,
	userID pulid.ID,
) (*holdreason.HoldReason, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("code", hr.Code),
		zap.String("buID", hr.BusinessUnitID.String()),
		zap.String("orgID", hr.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, hr); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, hr)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHoldReason,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Hold reason created"),
	)
	if err != nil {
		log.Error("failed to log hold reason creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	hr *holdreason.HoldReason,
	userID pulid.ID,
) (*holdreason.HoldReason, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("code", hr.Code),
		zap.String("buID", hr.BusinessUnitID.String()),
		zap.String("orgID", hr.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, hr); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetHoldReasonByIDRequest{
		ID:    hr.ID,
		OrgID: hr.OrganizationID,
		BuID:  hr.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, hr)
	if err != nil {
		log.Error("failed to update hold reason", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceHoldReason,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Hold reason updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log hold reason update", zap.Error(err))
	}

	return updatedEntity, nil
}
