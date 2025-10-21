package billingcontrol

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
	"github.com/emoss08/trenova/pkg/validator/billingcontrolvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.BillingControlRepository
	AuditService services.AuditService
	Validator    *billingcontrolvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.BillingControlRepository
	as   services.AuditService
	v    *billingcontrolvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.billingcontrol"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetBillingControlRequest,
) (*tenant.BillingControl, error) {
	return s.repo.GetByOrgID(ctx, req.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	bc *tenant.BillingControl,
	userID pulid.ID,
) (*tenant.BillingControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", bc.ID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, bc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByOrgID(ctx, bc.OrganizationID)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.Update(ctx, bc)
	if err != nil {
		log.Error("failed to update billing control", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceBillingControl,
			ResourceID:     entity.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(entity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: entity.ID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Billing control updated"),
		audit.WithDiff(original, entity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log billing control update", zap.Error(err))
	}

	return entity, nil
}
