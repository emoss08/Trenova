package edipayloadbuild

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ServiceFailureRepo servicefailureports.ServiceFailureRepository
	ShipmentSvc        shipmentserviceports.ShipmentReader
}

type Service struct {
	serviceFailureRepo servicefailureports.ServiceFailureRepository
	shipmentSvc        shipmentserviceports.ShipmentReader
}

func New(p Params) *Service {
	return &Service{
		serviceFailureRepo: p.ServiceFailureRepo,
		shipmentSvc:        p.ShipmentSvc,
	}
}
