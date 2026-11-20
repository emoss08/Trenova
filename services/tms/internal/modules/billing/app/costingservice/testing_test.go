package costingservice

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/zap"
)

type TestServiceParams struct {
	DB           kernelports.Transactor
	Repo         costingcontrolports.CostingControlRepository
	ActualsRepo  costingcontrolports.CostingActualsRepository
	PriceRepo    fuelsurchargeports.FuelIndexPriceRepository
	ShipmentRepo shipmentports.ShipmentRepository
	AuditService auditports.AuditService
	Fields       permissionports.FieldAccess
	Now          func() time.Time
}

func NewTestService(p TestServiceParams) *Service {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		l:            zap.NewNop(),
		db:           p.DB,
		repo:         p.Repo,
		actualsRepo:  p.ActualsRepo,
		priceRepo:    p.PriceRepo,
		shipmentRepo: p.ShipmentRepo,
		validator:    NewValidator(),
		auditService: p.AuditService,
		fields:       p.Fields,
		now:          now,
	}
}
