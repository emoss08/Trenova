package dispatchcontrol

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/dispatchcontrolvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.DispatchControlRepository
	AuditService services.AuditService
	Validator    *dispatchcontrolvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.DispatchControlRepository
	as   services.AuditService
	v    *dispatchcontrolvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.dispatchcontrol"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetDispatchControlRequest,
) (*tenant.DispatchControl, error) {
	return s.repo.GetByOrgID(ctx, req.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	entity *tenant.DispatchControl,
	userID pulid.ID,
) (*tenant.DispatchControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update dispatch control", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDispatchControl,
			ResourceID:     updatedEntity.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.ID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Dispatch control updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log dispatch control update", zap.Error(err))
	}

	return updatedEntity, nil
}
