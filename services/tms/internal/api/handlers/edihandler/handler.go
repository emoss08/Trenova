package edihandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ediservice.Service
	InboundService       *ediinboundservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service        *ediservice.Service
	inboundService *ediinboundservice.Service
	eh             *helpers.ErrorHandler
	pm             *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:        p.Service,
		inboundService: p.InboundService,
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
	h.registerSourceContextRoutes(catalog.Group("/source-context"))
	h.registerPartnerSettingsRoutes(catalog.Group("/partner-settings"))
	h.registerTemplateRoutes(api.Group("/templates"))
	h.registerDocumentProfileRoutes(api.Group("/document-profiles"))
	h.registerDocumentRoutes(api.Group("/documents"))
	h.registerMessageRoutes(api.Group("/messages"))
	h.registerInboundFileRoutes(api.Group("/inbound-files"))
	h.registerX12Routes(api.Group("/x12"))
	h.registerTestCaseRoutes(api.Group("/test-cases"))
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
