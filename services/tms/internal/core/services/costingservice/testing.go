package costingservice

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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
	Repo         repositories.CostingControlRepository
	ActualsRepo  repositories.CostingActualsRepository
	PriceRepo    repositories.FuelIndexPriceRepository
	ShipmentRepo repositories.ShipmentRepository
	AuditService services.AuditService
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
