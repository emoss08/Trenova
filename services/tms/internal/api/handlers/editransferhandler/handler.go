package editransferhandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editenderservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferchangeservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransferservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Setup                *edisetup.Service
	Transfers            *editransferservice.Service
	TenderChanges        *editenderchangeservice.Service
	TransferChanges      *editransferchangeservice.Service
	Tenders              *editenderservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	setup           *edisetup.Service
	transfers       *editransferservice.Service
	tenderChanges   *editenderchangeservice.Service
	transferChanges *editransferchangeservice.Service
	tenders         *editenderservice.Service
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		setup:           p.Setup,
		transfers:       p.Transfers,
		tenderChanges:   p.TenderChanges,
		transferChanges: p.TransferChanges,
		tenders:         p.Tenders,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/edi")

	api.POST(
		"/load-tenders/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.submitLoadTender,
	)
	api.POST(
		"/control-numbers/reset/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.resetControlNumber,
	)
	h.registerTransferRoutes(api.Group("/transfers"))
	h.registerShipmentLinkRoutes(api.Group("/shipment-links"))
	h.registerTransferChangeRoutes(api.Group("/transfer-changes"))
	h.registerTenderChangeRoutes(api.Group("/tender-changes"))
}
