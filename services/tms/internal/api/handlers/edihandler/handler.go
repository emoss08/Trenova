package edihandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edimessageview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Delivery             *edidelivery.Service
	Outbound             *edioutbound.Service
	Messages             *edimessageview.Service
	InboundService       *ediinboundservice.Service
	AS2Service           *ediinboundas2.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	delivery       *edidelivery.Service
	outbound       *edioutbound.Service
	messages       *edimessageview.Service
	inboundService *ediinboundservice.Service
	as2Service     *ediinboundas2.Service
	eh             *helpers.ErrorHandler
	pm             *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		delivery:       p.Delivery,
		outbound:       p.Outbound,
		messages:       p.Messages,
		inboundService: p.InboundService,
		as2Service:     p.AS2Service,
		eh:             p.ErrorHandler,
		pm:             p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/edi")

	h.registerDocumentRoutes(api.Group("/documents"))
	h.registerMessageRoutes(api.Group("/messages"))
	h.registerInboundFileRoutes(api.Group("/inbound-files"))
	h.registerX12Routes(api.Group("/x12"))
}
