package editendershipment

import (
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	ShipmentCommentRepo shipmentcommentports.ShipmentCommentRepository
	ShipmentTenderRepo  editenderports.EDIShipmentTenderRepository
	UserRepo            tenantports.UserRepository
}

type Service struct {
	shipmentCommentRepo shipmentcommentports.ShipmentCommentRepository
	shipmentTenderRepo  editenderports.EDIShipmentTenderRepository
	userRepo            tenantports.UserRepository
}

func New(p Params) *Service {
	return &Service{
		shipmentCommentRepo: p.ShipmentCommentRepo,
		shipmentTenderRepo:  p.ShipmentTenderRepo,
		userRepo:            p.UserRepo,
	}
}
