package edipayloadbuild

import (
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	OrderRepo          orderports.OrderRepository
	ServiceFailureRepo servicefailureports.ServiceFailureRepository
	ShipmentRepo       shipmentports.ShipmentRepository
	ShipmentSvc        shipmentserviceports.ShipmentReader
}

type Service struct {
	orderRepo          orderports.OrderRepository
	serviceFailureRepo servicefailureports.ServiceFailureRepository
	shipmentRepo       shipmentports.ShipmentRepository
	shipmentSvc        shipmentserviceports.ShipmentReader
}

func New(p Params) *Service {
	return &Service{
		orderRepo:          p.OrderRepo,
		serviceFailureRepo: p.ServiceFailureRepo,
		shipmentRepo:       p.ShipmentRepo,
		shipmentSvc:        p.ShipmentSvc,
	}
}
