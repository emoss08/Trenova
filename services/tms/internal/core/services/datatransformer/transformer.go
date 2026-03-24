package datatransformer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accounttype"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   repositories.DataEntryControlRepository
}

type Service struct {
	l    *zap.Logger
	repo repositories.DataEntryControlRepository
}

func New(p Params) services.DataTransformer {
	return &Service{
		l:    p.Logger.Named("service.datatransformer"),
		repo: p.Repo,
	}
}

func (s *Service) getControl(
	ctx context.Context,
	orgID, buID pulid.ID,
) (*dataentrycontrol.DataEntryControl, error) {
	return s.repo.GetOrCreate(ctx, orgID, buID)
}

func (s *Service) TransformFleetCode(ctx context.Context, entity *fleetcode.FleetCode) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Description = cleanText(entity.Description)

	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformEquipmentType(
	ctx context.Context,
	entity *equipmenttype.EquipmentType,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformAccessorialCharge(
	ctx context.Context,
	entity *accessorialcharge.AccessorialCharge,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformServiceType(
	ctx context.Context,
	entity *servicetype.ServiceType,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformShipmentType(
	ctx context.Context,
	entity *shipmenttype.ShipmentType,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformHazardousMaterial(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Name = applyCase(entity.Name, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformCommodity(
	ctx context.Context,
	entity *commodity.Commodity,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Name = applyCase(entity.Name, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformCustomer(
	ctx context.Context,
	entity *customer.Customer,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformAccountType(
	ctx context.Context,
	entity *accounttype.AccountType,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformGLAccount(
	ctx context.Context,
	entity *glaccount.GLAccount,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.AccountCode = cleanCode(entity.AccountCode)
	entity.AccountCode = applyCase(entity.AccountCode, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformLocation(
	ctx context.Context,
	entity *location.Location,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformFiscalYear(
	ctx context.Context,
	entity *fiscalyear.FiscalYear,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Name = cleanText(entity.Name)
	entity.Name = applyCase(entity.Name, ctrl.CodeCase)

	return nil
}

func (s *Service) TransformDocumentType(
	ctx context.Context,
	entity *documenttype.DocumentType,
) error {
	ctrl, err := s.getControl(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		s.l.Warn("failed to get data entry control, skipping transformation", zap.Error(err))
		return nil
	}

	entity.Code = cleanCode(entity.Code)
	entity.Code = applyCase(entity.Code, ctrl.CodeCase)

	return nil
}
