package costingservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Repo         costingcontrolports.CostingControlRepository
	ActualsRepo  costingcontrolports.CostingActualsRepository
	PriceRepo    fuelsurchargeports.FuelIndexPriceRepository
	ShipmentRepo shipmentports.ShipmentRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Fields       permissionports.FieldAccess
}

type Service struct {
	l            *zap.Logger
	db           kernelports.Transactor
	repo         costingcontrolports.CostingControlRepository
	actualsRepo  costingcontrolports.CostingActualsRepository
	priceRepo    fuelsurchargeports.FuelIndexPriceRepository
	shipmentRepo shipmentports.ShipmentRepository
	validator    *Validator
	auditService auditports.AuditService
	fields       permissionports.FieldAccess
	now          func() time.Time
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:            p.Logger.Named("service.costing"),
		db:           p.DB,
		repo:         p.Repo,
		actualsRepo:  p.ActualsRepo,
		priceRepo:    p.PriceRepo,
		shipmentRepo: p.ShipmentRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
		fields:       p.Fields,
		now:          time.Now,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *costingcontrolports.GetCostingControlRequest,
) (*costingcontrol.CostingControl, error) {
	return s.repo.GetByOrgID(ctx, req)
}
