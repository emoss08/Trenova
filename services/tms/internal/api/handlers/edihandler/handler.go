package edihandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ediservice.Service
	InboundService       *ediinboundservice.Service
	AS2Service           *ediinboundas2.Service
	PollService          *ediinboundpoll.Service
	Projects             *edilading.Projects
	LadingRuntime        *edilading.Runtime
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service        *ediservice.Service
	inboundService *ediinboundservice.Service
	as2Service     *ediinboundas2.Service
	pollService    *ediinboundpoll.Service
	projects       *edilading.Projects
	ladingRuntime  *edilading.Runtime
	eh             *helpers.ErrorHandler
	pm             *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:        p.Service,
		inboundService: p.InboundService,
		as2Service:     p.AS2Service,
		pollService:    p.PollService,
		projects:       p.Projects,
		ladingRuntime:  p.LadingRuntime,
		eh:             p.ErrorHandler,
		pm:             p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/edi")
	catalog := api.Group("/catalog")

	h.registerPartnerRoutes(api.Group("/partners"))
	h.registerMappingProfileRoutes(api.Group("/mapping-profiles"))
	h.registerConnectionRoutes(api.Group("/connections"))
	h.registerCommunicationProfileRoutes(api.Group("/communication-profiles"))
	h.registerDocumentTypeRoutes(catalog.Group("/document-types"))
	h.registerTransactionSetRoutes(catalog.Group("/transaction-sets"))
	h.registerLadingRoutes(api.Group("/lading"))
	h.registerFlowRoutes(api.Group("/flows"))
	h.registerDocumentRoutes(api.Group("/documents"))
	h.registerMessageRoutes(api.Group("/messages"))
	h.registerInboundFileRoutes(api.Group("/inbound-files"))
	h.registerX12Routes(api.Group("/x12"))
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
