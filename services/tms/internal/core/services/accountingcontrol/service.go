package accountingcontrol

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/accountingcontrolvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AccountingControlRepository
	AuditService services.AuditService
	Validator    *accountingcontrolvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.AccountingControlRepository
	as   services.AuditService
	v    *accountingcontrolvalidator.Validator
}

func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.shipmentcontrol"),
		repo: p.Repo,
		as:   p.AuditService,
		v:    p.Validator,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAccountingControlRequest,
) (*accounting.AccountingControl, error) {
	return s.repo.GetByOrgID(ctx, req.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	ac *accounting.AccountingControl,
	userID pulid.ID,
) (*accounting.AccountingControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", ac.ID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, ac); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByOrgID(ctx, ac.OrganizationID)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.Update(ctx, ac)
	if err != nil {
		log.Error("failed to update accounting control", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceAccountingControl,
			ResourceID:     entity.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(entity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: entity.ID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Accounting control updated"),
		audit.WithDiff(original, entity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log accounting control update", zap.Error(err))
	}

	return entity, nil
}
