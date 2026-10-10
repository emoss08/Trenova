package costingservice

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/fuelsurchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/zap"
)

func NewTestValidator() *Validator {
	return &Validator{
		controlValidator: validationframework.
			NewTenantedValidatorBuilder[*costingcontrol.CostingControl]().
			WithModelName("CostingControl").
			WithCustomRule(createGLActualsRule()).
			Build(),
		categoryValidator: validationframework.
			NewTenantedValidatorBuilder[*costingcontrol.CostCategory]().
			WithModelName("CostCategory").
			Build(),
	}
}

type TestServiceParams struct {
	Repo         costingcontrolports.CostingControlRepository
	ActualsRepo  costingcontrolports.CostingActualsRepository
	PriceRepo    fuelsurchargeports.FuelIndexPriceRepository
	ShipmentRepo shipmentports.ShipmentRepository
	AuditService auditports.AuditService
	Now          func() time.Time
}

func NewTestService(p TestServiceParams) *Service {
	now := p.Now
	if now == nil {
		now = time.Now
	}

	return &Service{
		l:            zap.NewNop(),
		repo:         p.Repo,
		actualsRepo:  p.ActualsRepo,
		priceRepo:    p.PriceRepo,
		shipmentRepo: p.ShipmentRepo,
		validator:    NewTestValidator(),
		auditService: p.AuditService,
		now:          now,
	}
}
