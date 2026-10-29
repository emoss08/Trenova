package edihandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundpoll"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/edimessageview"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioutbound"
	"github.com/emoss08/trenova/internal/modules/edi/app/edioverview"
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
	Delivery             *edidelivery.Service
	Outbound             *edioutbound.Service
	Messages             *edimessageview.Service
	Overview             *edioverview.Service
	Tenders              *editenderservice.Service
	InboundService       *ediinboundservice.Service
	AS2Service           *ediinboundas2.Service
	PollService          *ediinboundpoll.Service
	Projects             *edilading.Projects
	LadingRuntime        *edilading.Runtime
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	setup           *edisetup.Service
	transfers       *editransferservice.Service
	tenderChanges   *editenderchangeservice.Service
	transferChanges *editransferchangeservice.Service
	delivery        *edidelivery.Service
	outbound        *edioutbound.Service
	messages        *edimessageview.Service
	overview        *edioverview.Service
	tenders         *editenderservice.Service
	inboundService  *ediinboundservice.Service
	as2Service      *ediinboundas2.Service
	pollService     *ediinboundpoll.Service
	projects        *edilading.Projects
	ladingRuntime   *edilading.Runtime
	eh              *helpers.ErrorHandler
	pm              *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		setup:           p.Setup,
		transfers:       p.Transfers,
		tenderChanges:   p.TenderChanges,
		transferChanges: p.TransferChanges,
		delivery:        p.Delivery,
		outbound:        p.Outbound,
		messages:        p.Messages,
		overview:        p.Overview,
		tenders:         p.Tenders,
		inboundService:  p.InboundService,
		as2Service:      p.AS2Service,
		pollService:     p.PollService,
		projects:        p.Projects,
		ladingRuntime:   p.LadingRuntime,
		eh:              p.ErrorHandler,
		pm:              p.PermissionMiddleware,
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
