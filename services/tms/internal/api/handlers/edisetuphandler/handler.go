package edisetuphandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Setup                *edisetup.Service
	Overview             *edioverview.Service
	PollService          *ediinboundpoll.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	setup       *edisetup.Service
	overview    *edioverview.Service
	pollService *ediinboundpoll.Service
	eh          *helpers.ErrorHandler
	pm          *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		setup:       p.Setup,
		overview:    p.Overview,
		pollService: p.PollService,
		eh:          p.ErrorHandler,
		pm:          p.PermissionMiddleware,
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
}
