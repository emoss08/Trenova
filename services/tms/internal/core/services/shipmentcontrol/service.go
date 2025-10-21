package shipmentcontrol

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
	"github.com/emoss08/trenova/pkg/validator/shipmentcontrolvalidator"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.ShipmentControlRepository
	AuditService services.AuditService
	Validator    *shipmentcontrolvalidator.Validator
}

type Service struct {
	l    *zap.Logger
	repo repositories.ShipmentControlRepository
	as   services.AuditService
	v    *shipmentcontrolvalidator.Validator
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
	req repositories.GetShipmentControlRequest,
) (*tenant.ShipmentControl, error) {
	return s.repo.GetByOrgID(ctx, req.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	sc *tenant.ShipmentControl,
	userID pulid.ID,
) (*tenant.ShipmentControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", sc.ID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, sc); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByOrgID(ctx, sc.OrganizationID)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.Update(ctx, sc)
	if err != nil {
		log.Error("failed to update shipment control", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentControl,
			ResourceID:     entity.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(entity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: entity.ID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Shipment control updated"),
		audit.WithDiff(original, entity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log shipment control update", zap.Error(err))
	}

	return entity, nil
}
