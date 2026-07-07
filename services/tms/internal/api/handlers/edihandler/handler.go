package edihandler

import (
	"context"
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

type partnerRequest struct {
	Kind                       edi.PartnerKind `json:"kind"`
	Status                     string          `json:"status"`
	Code                       string          `json:"code"`
	Name                       string          `json:"name"`
	Description                string          `json:"description"`
	InternalOrganizationID     pulid.ID        `json:"internalOrganizationId"`
	EDIConnectionID            pulid.ID        `json:"ediConnectionId"`
	CustomerID                 pulid.ID        `json:"customerId"`
	DefaultTransportID         pulid.ID        `json:"defaultTransportId"`
	DefaultMappingProfileID    pulid.ID        `json:"defaultMappingProfileId"`
	DefaultValidationProfileID pulid.ID        `json:"defaultValidationProfileId"`
	Timezone                   string          `json:"timezone"`
	Country                    string          `json:"country"`
	ContactName                string          `json:"contactName"`
	ContactEmail               string          `json:"contactEmail"`
	ContactPhone               string          `json:"contactPhone"`
	EnabledForInbound          *bool           `json:"enabledForInbound"`
	EnabledForOutbound         *bool           `json:"enabledForOutbound"`
	Settings                   map[string]any  `json:"settings"`
	Version                    int64           `json:"version"`
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
	h.registerTransferRoutes(api.Group("/transfers"))
	h.registerShipmentLinkRoutes(api.Group("/shipment-links"))
	h.registerTransferChangeRoutes(api.Group("/transfer-changes"))
	h.registerTenderChangeRoutes(api.Group("/tender-changes"))
}

func (h *Handler) registerPartnerRoutes(partners *gin.RouterGroup) {
	partners.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartners,
	)
	partners.GET(
		"/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerOptions,
	)
	partners.POST(
		"/internal-pairs/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createInternalPartnerPair,
	)
	partners.GET(
		"/:partnerID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartner,
	)
	partners.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createPartner,
	)
	partners.PUT(
		"/:partnerID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updatePartner,
	)
	partners.GET(
		"/:partnerID/mapping-profile/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfile,
	)
	partners.PUT(
		"/:partnerID/mapping-profile/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateMappingProfile,
	)
	partners.DELETE(
		"/:partnerID/mapping-profile/items/:mappingItemID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.deleteMappingItem,
	)
}

func (h *Handler) registerMappingProfileRoutes(mappingProfiles *gin.RouterGroup) {
	selectOptions := mappingProfiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectMappingProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfileOption,
	)
	mappingProfiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listMappingProfiles,
	)
	mappingProfiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMappingProfileByID,
	)
	mappingProfiles.PUT(
		"/:profileID/items/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateMappingProfileItems,
	)
	mappingProfiles.DELETE(
		"/:profileID/items/:mappingItemID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.deleteMappingProfileItem,
	)
}

func (h *Handler) registerConnectionRoutes(connections *gin.RouterGroup) {
	connections.GET(
		"/",
		// h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listConnections,
	)
	connections.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createConnection,
	)
	connections.GET(
		"/:connectionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getConnection,
	)
	connections.POST(
		"/:connectionID/accept/",
		// h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.acceptConnection,
	)
	connections.POST(
		"/:connectionID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectConnection,
	)
	connections.POST(
		"/:connectionID/suspend/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.suspendConnection,
	)
	connections.POST(
		"/:connectionID/revoke/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.revokeConnection,
	)
}

func (h *Handler) registerCommunicationProfileRoutes(profiles *gin.RouterGroup) {
	selectOptions := profiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectCommunicationProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getCommunicationProfileOption,
	)
	profiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listCommunicationProfiles,
	)
	profiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createCommunicationProfile,
	)
	profiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getCommunicationProfile,
	)
	profiles.PUT(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateCommunicationProfile,
	)
}

func (h *Handler) registerDocumentTypeRoutes(documentTypes *gin.RouterGroup) {
	selectOptions := documentTypes.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectDocumentTypeOptions,
	)
	selectOptions.GET(
		"/:documentTypeID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getDocumentTypeOption,
	)
	documentTypes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listDocumentTypes,
	)
}

func (h *Handler) registerSourceContextRoutes(sourceContext *gin.RouterGroup) {
	sourceContext.GET(
		"/schemas/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listSourceContextSchemas,
	)
	sourceContext.GET(
		"/schemas/:schemaID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getSourceContextSchema,
	)
	sourceContext.GET(
		"/schemas/:schemaID/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listSourceContextSchemaFields,
	)
	sourceContext.GET(
		"/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.searchSourceContextFields,
	)
	sourceContext.GET(
		"/fields/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectSourceContextFieldOptions,
	)
}

func (h *Handler) registerPartnerSettingsRoutes(partnerSettings *gin.RouterGroup) {
	partnerSettings.GET(
		"/schemas/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerSettingSchemas,
	)
	partnerSettings.GET(
		"/schemas/:schemaID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerSettingSchema,
	)
	partnerSettings.GET(
		"/schemas/:schemaID/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerSettingSchemaFields,
	)
	partnerSettings.GET(
		"/fields/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.searchPartnerSettingFields,
	)
	partnerSettings.GET(
		"/fields/select-options/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerSettingFieldOptions,
	)
	partnerSettings.POST(
		"/validate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.validatePartnerSettings,
	)
}

func (h *Handler) registerTemplateRoutes(templates *gin.RouterGroup) {
	selectOptions := templates.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectTemplateOptions,
	)
	selectOptions.GET(
		"/:templateID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplateOption,
	)
	templates.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplates,
	)
	templates.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createTemplate,
	)
	templates.GET(
		"/:templateID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplate,
	)
	templates.PUT(
		"/:templateID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTemplate,
	)
	templates.POST(
		"/:templateID/draft/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createDraftVersion,
	)
	templates.GET(
		"/:templateID/versions/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplateVersions,
	)
	templates.GET(
		"/:templateID/versions/:versionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTemplateVersion,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTemplateVersion,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/segments/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.replaceTemplateSegments,
	)
	templates.GET(
		"/:templateID/versions/:versionID/script-libraries/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTemplateScriptLibraries,
	)
	templates.PUT(
		"/:templateID/versions/:versionID/script-libraries/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.replaceTemplateScriptLibraries,
	)
	templates.POST(
		"/:templateID/versions/:versionID/validate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.validateTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/certify/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.certifyTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/activate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.activateTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/archive/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.archiveTemplateVersion,
	)
	templates.POST(
		"/:templateID/versions/:versionID/rollback/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rollbackTemplateVersion,
	)
}

func (h *Handler) registerDocumentProfileRoutes(documentProfiles *gin.RouterGroup) {
	selectOptions := documentProfiles.Group("/select-options")
	selectOptions.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.selectPartnerDocumentProfileOptions,
	)
	selectOptions.GET(
		"/:profileID",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerDocumentProfileOption,
	)
	documentProfiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listPartnerDocumentProfiles,
	)
	documentProfiles.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createPartnerDocumentProfile,
	)
	documentProfiles.GET(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getPartnerDocumentProfile,
	)
	documentProfiles.PUT(
		"/:profileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updatePartnerDocumentProfile,
	)
}

func (h *Handler) registerDocumentRoutes(documents *gin.RouterGroup) {
	documents.POST(
		"/preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.previewDocument,
	)
	documents.POST(
		"/generate/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.generateDocument,
	)
}

func (h *Handler) registerMessageRoutes(messages *gin.RouterGroup) {
	messages.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listMessages,
	)
	messages.GET(
		"/:messageID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMessage,
	)
	messages.GET(
		"/:messageID/inspect/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.inspectMessage,
	)
	messages.POST(
		"/:messageID/retry-delivery/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.retryMessageDelivery,
	)
}

func (h *Handler) registerInboundFileRoutes(inboundFiles *gin.RouterGroup) {
	inboundFiles.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listInboundFiles,
	)
	inboundFiles.GET(
		"/:fileID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getInboundFile,
	)
	inboundFiles.POST(
		"/:fileID/reprocess/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.reprocessInboundFile,
	)
}

func (h *Handler) registerX12Routes(x12 *gin.RouterGroup) {
	x12.POST(
		"/inspect/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.inspectX12,
	)
}

func (h *Handler) registerTestCaseRoutes(testCases *gin.RouterGroup) {
	testCases.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTestCases,
	)
	testCases.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpCreate),
		h.createTestCase,
	)
	testCases.GET(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTestCase,
	)
	testCases.PUT(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.updateTestCase,
	)
	testCases.DELETE(
		"/:testCaseID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpDelete),
		h.deleteTestCase,
	)
	testCases.POST(
		"/:testCaseID/preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.previewTestCase,
	)
}

func (h *Handler) registerTransferRoutes(transfers *gin.RouterGroup) {
	transfers.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTransfers,
	)
	transfers.GET(
		"/:transferID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTransfer,
	)
	transfers.GET(
		"/:transferID/mapping-preview/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.mappingPreview,
	)
	transfers.POST(
		"/:transferID/approve/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.approveTransfer,
	)
	transfers.POST(
		"/:transferID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTransfer,
	)
	transfers.POST(
		"/:transferID/cancel/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.cancelTransfer,
	)
	transfers.POST(
		"/:transferID/expire/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.expireTransfer,
	)
}

func (h *Handler) registerShipmentLinkRoutes(links *gin.RouterGroup) {
	links.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listShipmentLinks,
	)
	links.GET(
		"/:linkID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getShipmentLink,
	)
}

func (h *Handler) registerTransferChangeRoutes(changes *gin.RouterGroup) {
	changes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTransferChanges,
	)
	changes.GET(
		"/:changeID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTransferChange,
	)
	changes.POST(
		"/:changeID/apply/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.applyTransferChange,
	)
	changes.POST(
		"/:changeID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTransferChange,
	)
}

func (h *Handler) registerTenderChangeRoutes(changes *gin.RouterGroup) {
	changes.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listTenderChanges,
	)
	changes.GET(
		"/:changeID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getTenderChange,
	)
	changes.POST(
		"/:changeID/apply/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.applyTenderChange,
	)
	changes.POST(
		"/:changeID/reject/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.rejectTenderChange,
	)
}

func (h *Handler) listPartners(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIPartner], error) {
		return h.service.ListPartners(
			c.Request.Context(),
			&repositories.ListEDIPartnersRequest{Filter: req},
		)
	})
}

func (h *Handler) selectPartnerOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIPartner], error) {
		return h.service.SelectPartnerOptions(
			c.Request.Context(),
			&repositories.EDIPartnerSelectOptionsRequest{
				SelectQueryRequest: req,
				Kind:               edi.PartnerKind(helpers.QueryString(c, "kind", "")),
				EnabledForOutbound: helpers.QueryBool(c, "enabledForOutbound", false),
			},
		)
	})
}

func (h *Handler) getPartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetPartner(c.Request.Context(), repositories.GetEDIPartnerByIDRequest{
		ID: partnerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createPartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(partnerRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := req.toEntity(true)
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	created, err := h.service.CreatePartner(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) createInternalPartnerPair(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateInternalPartnerPairRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	pair, err := h.service.CreateInternalPartnerPair(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pair)
}

func (h *Handler) updatePartner(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(partnerRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := req.toEntity(false)
	entity.ID = partnerID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	updated, err := h.service.UpdatePartner(
		c.Request.Context(),
		entity,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (r *partnerRequest) toEntity(defaultEnabled bool) *edi.EDIPartner {
	enabledForInbound := defaultEnabled
	if r.EnabledForInbound != nil {
		enabledForInbound = *r.EnabledForInbound
	}
	enabledForOutbound := defaultEnabled
	if r.EnabledForOutbound != nil {
		enabledForOutbound = *r.EnabledForOutbound
	}

	return &edi.EDIPartner{
		Kind:                       r.Kind,
		Status:                     domaintypes.Status(r.Status),
		Code:                       r.Code,
		Name:                       r.Name,
		Description:                r.Description,
		InternalOrganizationID:     r.InternalOrganizationID,
		EDIConnectionID:            r.EDIConnectionID,
		CustomerID:                 r.CustomerID,
		DefaultTransportID:         r.DefaultTransportID,
		DefaultMappingProfileID:    r.DefaultMappingProfileID,
		DefaultValidationProfileID: r.DefaultValidationProfileID,
		Timezone:                   r.Timezone,
		Country:                    r.Country,
		ContactName:                r.ContactName,
		ContactEmail:               r.ContactEmail,
		ContactPhone:               r.ContactPhone,
		EnabledForInbound:          enabledForInbound,
		EnabledForOutbound:         enabledForOutbound,
		Settings:                   r.Settings,
		Version:                    r.Version,
	}
}

func (h *Handler) getMappingProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.service.GetMappingProfile(
		c.Request.Context(),
		repositories.GetMappingProfileRequest{
			PartnerID: partnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) updateMappingProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := struct {
		Items []*edi.EDIMappingProfileItem `json:"items"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.service.SaveMappingProfile(
		c.Request.Context(),
		&repositories.SaveMappingItemsRequest{
			PartnerID: partnerID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			ActorID: authCtx.UserID,
			Items:   req.Items,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) deleteMappingItem(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	partnerID, err := pulid.MustParse(c.Param("partnerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	mappingItemID, err := pulid.MustParse(c.Param("mappingItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteMappingItem(
		c.Request.Context(),
		repositories.DeleteMappingItemRequest{
			PartnerID:     partnerID,
			MappingItemID: mappingItemID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) listMappingProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIMappingProfile], error) {
		return h.service.ListMappingProfiles(
			c.Request.Context(),
			&repositories.ListEDIMappingProfilesRequest{Filter: req},
		)
	})
}

func (h *Handler) selectMappingProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIMappingProfile], error) {
			return h.service.SelectMappingProfileOptions(
				c.Request.Context(),
				&repositories.EDIMappingProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					PartnerID:          helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) getMappingProfileOption(c *gin.Context) {
	h.getMappingProfileByID(c)
}

func (h *Handler) getMappingProfileByID(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.service.GetMappingProfileByID(
		c.Request.Context(),
		repositories.GetMappingProfileByIDRequest{
			ProfileID: profileID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) updateMappingProfileItems(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := struct {
		Items []*edi.EDIMappingProfileItem `json:"items"`
	}{}
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.service.SaveMappingProfileItems(
		c.Request.Context(),
		&repositories.SaveMappingProfileItemsRequest{
			ProfileID: profileID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			ActorID: authCtx.UserID,
			Items:   req.Items,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, items)
}

func (h *Handler) deleteMappingProfileItem(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	mappingItemID, err := pulid.MustParse(c.Param("mappingItemID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteMappingProfileItem(
		c.Request.Context(),
		repositories.DeleteMappingProfileItemRequest{
			ProfileID:     profileID,
			MappingItemID: mappingItemID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) listConnections(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIConnection], error) {
		return h.service.ListConnections(
			c.Request.Context(),
			&repositories.ListEDIConnectionsRequest{Filter: req},
		)
	})
}

func (h *Handler) createConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateEDIConnectionRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	connection, err := h.service.CreateConnection(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, connection)
}

func (h *Handler) getConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	connectionID, err := pulid.MustParse(c.Param("connectionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	connection, err := h.service.GetConnection(
		c.Request.Context(),
		repositories.GetEDIConnectionByIDRequest{
			ID: connectionID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, connection)
}

func (h *Handler) acceptConnection(c *gin.Context) {
	h.connectionAction(c, h.service.AcceptConnection)
}

func (h *Handler) rejectConnection(c *gin.Context) {
	h.connectionAction(c, h.service.RejectConnection)
}

func (h *Handler) suspendConnection(c *gin.Context) {
	h.connectionAction(c, h.service.SuspendConnection)
}

func (h *Handler) revokeConnection(c *gin.Context) {
	h.connectionAction(c, h.service.RevokeConnection)
}

func (h *Handler) connectionAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.EDIConnectionActionRequest,
		*services.RequestActor,
	) (*edi.EDIConnection, error),
) {
	authCtx := authctx.GetAuthContext(c)
	connectionID, err := pulid.MustParse(c.Param("connectionID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.EDIConnectionActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ConnectionID = connectionID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	connection, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, connection)
}

func (h *Handler) listCommunicationProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			return h.service.ListCommunicationProfiles(
				c.Request.Context(),
				&repositories.ListEDICommunicationProfilesRequest{Filter: req},
			)
		},
	)
}

func (h *Handler) selectCommunicationProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDICommunicationProfile], error) {
			return h.service.SelectCommunicationProfileOptions(
				c.Request.Context(),
				&repositories.EDICommunicationProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					Status:             domaintypes.Status(helpers.QueryString(c, "status", "")),
					Method:             edi.ConnectionMethod(helpers.QueryString(c, "method", "")),
					PartnerID:          helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) createCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.UpsertEDICommunicationProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	profile, err := h.service.CreateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *Handler) getCommunicationProfileOption(c *gin.Context) {
	h.getCommunicationProfile(c)
}

func (h *Handler) getCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	profile, err := h.service.GetCommunicationProfile(
		c.Request.Context(),
		repositories.GetEDICommunicationProfileByIDRequest{
			ID: profileID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) updateCommunicationProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.UpsertEDICommunicationProfileRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ProfileID = profileID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	profile, err := h.service.UpdateCommunicationProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *Handler) listDocumentTypes(c *gin.Context) {
	entities, err := h.service.ListDocumentTypes(
		c.Request.Context(),
		repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
			TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
			Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
			Status:         edi.DocumentStatus(helpers.QueryString(c, "status", "")),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entities)
}

func (h *Handler) selectDocumentTypeOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIDocumentType], error) {
			return h.service.SelectDocumentTypeOptions(
				c.Request.Context(),
				&repositories.EDIDocumentTypeSelectOptionsRequest{
					SelectQueryRequest: req,
					Standard:           edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getDocumentTypeOption(c *gin.Context) {
	documentTypeID, err := pulid.MustParse(c.Param("documentTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entities, err := h.service.ListDocumentTypes(
		c.Request.Context(),
		repositories.ListEDIDocumentTypesRequest{},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	for _, entity := range entities {
		if entity.ID == documentTypeID {
			c.JSON(http.StatusOK, entity)
			return
		}
	}
	h.eh.HandleError(c, errortypes.NewNotFoundError("EDI document type not found"))
}

func (h *Handler) listSourceContextSchemas(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextSchema], error) {
			return h.service.ListSourceContextSchemas(
				c.Request.Context(),
				&repositories.ListEDISourceContextSchemasRequest{
					Filter:   req,
					Standard: edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					X12Version:    helpers.QueryString(c, "x12Version", ""),
					ContextKey:    helpers.QueryString(c, "contextKey", ""),
					SchemaVersion: helpers.QueryInt64(c, "schemaVersion", 0),
					Status: edi.SourceContextFieldStatus(
						helpers.QueryString(c, "status", ""),
					),
				},
			)
		},
	)
}

func (h *Handler) getSourceContextSchema(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	schema, err := h.service.GetSourceContextSchema(
		c.Request.Context(),
		repositories.GetEDISourceContextSchemaRequest{
			ID:         schemaID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, schema)
}

func (h *Handler) listSourceContextSchemaFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			fieldReq := h.sourceContextFieldRequest(c, req)
			fieldReq.SchemaID = schemaID
			return h.service.ListSourceContextFields(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) searchSourceContextFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			return h.service.SearchSourceContextFields(
				c.Request.Context(),
				h.sourceContextFieldRequest(c, req),
			)
		},
	)
}

func (h *Handler) selectSourceContextFieldOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	fieldReq := h.sourceContextFieldRequest(c, queryOptionsFromSelect(req))

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDISourceContextField], error) {
			return h.service.SelectSourceContextFieldOptions(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) sourceContextFieldRequest(
	c *gin.Context,
	req *pagination.QueryOptions,
) *repositories.ListEDISourceContextFieldsRequest {
	fieldReq := &repositories.ListEDISourceContextFieldsRequest{
		Filter: req,
		Status: edi.SourceContextFieldStatus(
			helpers.QueryString(c, "status", ""),
		),
		Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
		TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
		Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
		SourceKind:     edi.SourceContextKind(helpers.QueryString(c, "sourceKind", "")),
		PathPrefix:     helpers.QueryString(c, "pathPrefix", ""),
	}
	if _, ok := c.GetQuery("repeated"); ok {
		repeated := helpers.QueryBool(c, "repeated", false)
		fieldReq.Repeated = &repeated
	}
	if schemaID := helpers.QueryPulid(c, "schemaId"); schemaID.IsNotNil() {
		fieldReq.SchemaID = schemaID
	}
	return fieldReq
}

func (h *Handler) listPartnerSettingSchemas(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingSchema], error) {
			return h.service.ListPartnerSettingSchemas(
				c.Request.Context(),
				&repositories.ListEDIPartnerSettingSchemasRequest{
					Filter:   req,
					Standard: edi.EDIStandard(helpers.QueryString(c, "standard", "")),
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
					X12Version:     helpers.QueryString(c, "x12Version", ""),
					DocumentTypeID: helpers.QueryPulid(c, "documentTypeId"),
					SchemaVersion:  helpers.QueryInt64(c, "schemaVersion", 0),
					Status:         edi.PartnerSettingStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getPartnerSettingSchema(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	schema, err := h.service.GetPartnerSettingSchema(
		c.Request.Context(),
		repositories.GetEDIPartnerSettingSchemaRequest{
			ID:         schemaID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, schema)
}

func (h *Handler) listPartnerSettingSchemaFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	schemaID, err := pulid.MustParse(c.Param("schemaID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			fieldReq := h.partnerSettingFieldRequest(c, req)
			fieldReq.SchemaID = schemaID
			return h.service.ListPartnerSettingFields(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) searchPartnerSettingFields(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			return h.service.SearchPartnerSettingFields(
				c.Request.Context(),
				h.partnerSettingFieldRequest(c, req),
			)
		},
	)
}

func (h *Handler) selectPartnerSettingFieldOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)
	fieldReq := h.partnerSettingFieldRequest(c, queryOptionsFromSelect(req))

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerSettingField], error) {
			return h.service.SelectPartnerSettingFieldOptions(c.Request.Context(), fieldReq)
		},
	)
}

func (h *Handler) partnerSettingFieldRequest(
	c *gin.Context,
	req *pagination.QueryOptions,
) *repositories.ListEDIPartnerSettingFieldsRequest {
	fieldReq := &repositories.ListEDIPartnerSettingFieldsRequest{
		Filter:         req,
		Standard:       edi.EDIStandard(helpers.QueryString(c, "standard", "")),
		TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
		Direction:      edi.DocumentDirection(helpers.QueryString(c, "direction", "")),
		Status:         edi.PartnerSettingStatus(helpers.QueryString(c, "status", "")),
		PathPrefix:     helpers.QueryString(c, "pathPrefix", ""),
		GroupKey:       helpers.QueryString(c, "groupKey", ""),
	}
	if _, ok := c.GetQuery("required"); ok {
		required := helpers.QueryBool(c, "required", false)
		fieldReq.Required = &required
	}
	if _, ok := c.GetQuery("secret"); ok {
		secret := helpers.QueryBool(c, "secret", false)
		fieldReq.Secret = &secret
	}
	if schemaID := helpers.QueryPulid(c, "schemaId"); schemaID.IsNotNil() {
		fieldReq.SchemaID = schemaID
	}
	return fieldReq
}

func (h *Handler) validatePartnerSettings(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.ValidatePartnerSettingsRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	diagnostics, err := h.service.ValidatePartnerSettings(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"diagnostics": diagnostics})
}

func (h *Handler) listTemplates(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITemplate], error) {
		return h.service.ListTemplates(
			c.Request.Context(),
			&repositories.ListEDITemplatesRequest{
				Filter:         req,
				TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
				Direction: edi.DocumentDirection(
					helpers.QueryString(c, "direction", ""),
				),
				Status: edi.TemplateStatus(helpers.QueryString(c, "status", "")),
			},
		)
	})
}

func (h *Handler) selectTemplateOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDITemplate], error) {
			return h.service.SelectTemplateOptions(
				c.Request.Context(),
				&repositories.EDITemplateSelectOptionsRequest{
					SelectQueryRequest: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status: edi.TemplateStatus(helpers.QueryString(c, "status", "")),
				},
			)
		},
	)
}

func (h *Handler) getTemplateOption(c *gin.Context) {
	h.getTemplate(c)
}

func (h *Handler) createTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.CreateEDITemplateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	created, err := h.service.CreateTemplate(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	template, err := h.service.GetTemplate(
		c.Request.Context(),
		repositories.GetEDITemplateByIDRequest{
			ID:         templateID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, template)
}

func (h *Handler) updateTemplate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpdateEDITemplateRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	updated, err := h.service.UpdateTemplate(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) createDraftVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.CreateEDITemplateDraftRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.TemplateID = templateID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.CreateDraftVersion(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, version)
}

func (h *Handler) listTemplateVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	versions, err := h.service.ListTemplateVersions(
		c.Request.Context(),
		repositories.ListEDITemplateVersionsRequest{
			TemplateID: templateID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

func (h *Handler) getTemplateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := h.service.GetTemplateVersion(
		c.Request.Context(),
		repositories.GetEDITemplateVersionByIDRequest{
			TemplateID: templateID,
			VersionID:  versionID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) updateTemplateVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpdateEDITemplateVersionRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.UpdateDraftVersion(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) replaceTemplateSegments(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.ReplaceEDITemplateSegmentsRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.ReplaceDraftSegments(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) listTemplateScriptLibraries(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	libraries, err := h.service.ListTemplateScriptLibraries(
		c.Request.Context(),
		repositories.ListEDITemplateScriptLibrariesRequest{
			TemplateID: templateID,
			VersionID:  versionID,
			TenantInfo: tenantInfoFromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, libraries)
}

func (h *Handler) replaceTemplateScriptLibraries(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.ReplaceEDITemplateScriptLibrariesRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	version, err := h.service.ReplaceDraftScriptLibraries(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) validateTemplateVersion(c *gin.Context) {
	req, err := h.templateActionRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	diagnostics, err := h.service.ValidateTemplateVersion(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"diagnostics": diagnostics})
}

func (h *Handler) certifyTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.CertifyTemplateVersion)
}

func (h *Handler) activateTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.ActivateTemplateVersion)
}

func (h *Handler) archiveTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.ArchiveTemplateVersion)
}

func (h *Handler) rollbackTemplateVersion(c *gin.Context) {
	h.templateVersionAction(c, h.service.RollbackTemplateVersion)
}

func (h *Handler) templateVersionAction(
	c *gin.Context,
	fn func(context.Context, *ediservice.EDIActionNotesRequest, *services.RequestActor) (*edi.EDITemplateVersion, error),
) {
	authCtx := authctx.GetAuthContext(c)
	req, err := h.templateActionRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, version)
}

func (h *Handler) templateActionRequest(c *gin.Context) (*ediservice.EDIActionNotesRequest, error) {
	authCtx := authctx.GetAuthContext(c)
	templateID, versionID, err := h.templateVersionIDs(c)
	if err != nil {
		return nil, err
	}
	req := new(ediservice.EDIActionNotesRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			return nil, err
		}
	}
	req.TemplateID = templateID
	req.VersionID = versionID
	req.TenantInfo = tenantInfoFromAuth(authCtx)
	return req, nil
}

func (h *Handler) templateVersionIDs(c *gin.Context) (
	templateID pulid.ID,
	versionID pulid.ID,
	err error,
) {
	templateID, err = pulid.MustParse(c.Param("templateID"))
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	versionID, err = pulid.MustParse(c.Param("versionID"))
	if err != nil {
		return pulid.Nil, pulid.Nil, err
	}
	return templateID, versionID, nil
}

func tenantInfoFromAuth(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func queryOptionsFromSelect(req *pagination.SelectQueryRequest) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: req.TenantInfo,
		Pagination: req.Pagination,
		Query:      req.Query,
	}
}

func (h *Handler) listPartnerDocumentProfiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			return h.service.ListPartnerDocumentProfiles(
				c.Request.Context(),
				&repositories.ListEDIPartnerDocumentProfilesRequest{
					Filter: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
					PartnerID: helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) selectPartnerDocumentProfileOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
			return h.service.SelectPartnerDocumentProfileOptions(
				c.Request.Context(),
				&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
					SelectQueryRequest: req,
					TransactionSet: edi.TransactionSet(
						helpers.QueryString(c, "transactionSet", ""),
					),
					Direction: edi.DocumentDirection(
						helpers.QueryString(c, "direction", ""),
					),
					Status:    edi.DocumentStatus(helpers.QueryString(c, "status", "")),
					PartnerID: helpers.QueryPulid(c, "partnerId"),
				},
			)
		},
	)
}

func (h *Handler) getPartnerDocumentProfileOption(c *gin.Context) {
	h.getPartnerDocumentProfile(c)
}

func (h *Handler) getPartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.GetPartnerDocumentProfile(
		c.Request.Context(),
		repositories.GetEDIPartnerDocumentProfileByIDRequest{
			ID: profileID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createPartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.UpsertEDIPartnerDocumentProfileRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	created, err := h.service.UpsertPartnerDocumentProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) updatePartnerDocumentProfile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	profileID, err := pulid.MustParse(c.Param("profileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(ediservice.UpsertEDIPartnerDocumentProfileRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.ProfileID = profileID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	updated, err := h.service.UpsertPartnerDocumentProfile(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) previewDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.PreviewEDIDocumentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	preview, err := h.service.PreviewDocument(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) generateDocument(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.GenerateEDIDocumentRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.GeneratedByID = authCtx.UserID
	message, err := h.service.GenerateDocument(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

func (h *Handler) listMessages(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	partnerID, _ := pulid.MustParse(helpers.QueryString(c, "partnerId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIMessage], error) {
		return h.service.ListMessages(
			c.Request.Context(),
			&repositories.ListEDIMessagesRequest{
				Filter:         req,
				TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
				Direction: edi.DocumentDirection(
					helpers.QueryString(c, "direction", ""),
				),
				PartnerID:     partnerID,
				Status:        edi.MessageStatus(helpers.QueryString(c, "status", "")),
				Query:         helpers.QueryStringTrimmed(c, "query", ""),
				GeneratedFrom: helpers.QueryInt64(c, "generatedFrom", 0),
				GeneratedTo:   helpers.QueryInt64(c, "generatedTo", 0),
			},
		)
	})
}

func (h *Handler) getMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.service.GetMessage(
		c.Request.Context(),
		repositories.GetEDIMessageByIDRequest{
			ID: messageID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, message)
}

func (h *Handler) inspectMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	inspection, err := h.service.InspectMessage(
		c.Request.Context(),
		repositories.GetEDIMessageByIDRequest{
			ID: messageID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, inspection)
}

func (h *Handler) listInboundFiles(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	partnerID, _ := pulid.MustParse(helpers.QueryString(c, "partnerId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIInboundFile], error) {
		return h.inboundService.ListInboundFiles(
			c.Request.Context(),
			&repositories.ListEDIInboundFilesRequest{
				Filter:    req,
				Status:    edi.InboundFileStatus(helpers.QueryString(c, "status", "")),
				PartnerID: partnerID,
			},
		)
	})
}

func (h *Handler) getInboundFile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fileID, err := pulid.MustParse(c.Param("fileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	file, err := h.inboundService.GetInboundFile(
		c.Request.Context(),
		repositories.GetEDIInboundFileByIDRequest{
			ID: fileID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			IncludeMessages: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, file)
}

func (h *Handler) reprocessInboundFile(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fileID, err := pulid.MustParse(c.Param("fileID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	file, err := h.inboundService.ProcessInboundFile(
		c.Request.Context(),
		&ediinboundservice.ProcessInboundFileRequest{
			FileID: fileID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Reprocess: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, file)
}

func (h *Handler) retryMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.service.RetryMessageDelivery(
		c.Request.Context(),
		&ediservice.RetryMessageDeliveryRequest{
			MessageID: messageID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, message)
}

func (h *Handler) inspectX12(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.InspectX12Request)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	inspection, err := h.service.InspectX12(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, inspection)
}

func (h *Handler) listTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	profileID, _ := pulid.MustParse(helpers.QueryString(c, "partnerDocumentProfileId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITestCase], error) {
		return h.service.ListTestCases(
			c.Request.Context(),
			&repositories.ListEDITestCasesRequest{
				Filter:                   req,
				PartnerDocumentProfileID: profileID,
			},
		)
	})
}

type testCaseRequest struct {
	PartnerDocumentProfileID pulid.ID            `json:"partnerDocumentProfileId"`
	Name                     string              `json:"name"`
	Description              string              `json:"description"`
	Payload                  edi.DocumentPayload `json:"payload"`
	ExpectedWarnings         int                 `json:"expectedWarnings"`
	ExpectedErrors           int                 `json:"expectedErrors"`
	Version                  int64               `json:"version"`
}

func (r *testCaseRequest) toServiceRequest(
	testCaseID pulid.ID,
	tenantInfo pagination.TenantInfo,
) *ediservice.SaveEDITestCaseRequest {
	return &ediservice.SaveEDITestCaseRequest{
		TenantInfo:               tenantInfo,
		ID:                       testCaseID,
		PartnerDocumentProfileID: r.PartnerDocumentProfileID,
		Name:                     r.Name,
		Description:              r.Description,
		Payload:                  r.Payload,
		ExpectedWarnings:         r.ExpectedWarnings,
		ExpectedErrors:           r.ExpectedErrors,
		Version:                  r.Version,
	}
}

func (h *Handler) createTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(testCaseRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	created, err := h.service.CreateTestCase(
		c.Request.Context(),
		req.toServiceRequest(pulid.Nil, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) getTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	testCase, err := h.service.GetTestCase(
		c.Request.Context(),
		repositories.GetEDITestCaseByIDRequest{
			ID: testCaseID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, testCase)
}

func (h *Handler) updateTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(testCaseRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	updated, err := h.service.UpdateTestCase(
		c.Request.Context(),
		req.toServiceRequest(testCaseID, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if err = h.service.DeleteTestCase(
		c.Request.Context(),
		repositories.DeleteEDITestCaseRequest{
			ID: testCaseID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) previewTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	preview, err := h.service.PreviewTestCase(
		c.Request.Context(),
		testCaseID,
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, preview)
}

func (h *Handler) submitLoadTender(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.SubmitLoadTenderRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.service.SubmitLoadTender(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, transfer)
}

func (h *Handler) listInboundTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITransfer], error) {
		return h.service.ListInboundTransfers(
			c.Request.Context(),
			&repositories.ListEDITransfersRequest{Filter: req},
		)
	})
}

func (h *Handler) listTransfers(c *gin.Context) {
	direction := c.Query("direction")
	switch direction {
	case "inbound":
		h.listInboundTransfers(c)
	case "outbound":
		h.listOutboundTransfers(c)
	default:
		h.eh.HandleError(c, errortypes.NewValidationError(
			"direction",
			errortypes.ErrInvalid,
			"Direction must be inbound or outbound",
		))
	}
}

func (h *Handler) listOutboundTransfers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDITransfer], error) {
		return h.service.ListOutboundTransfers(
			c.Request.Context(),
			&repositories.ListEDITransfersRequest{Filter: req},
		)
	})
}

func (h *Handler) getTransfer(c *gin.Context) {
	h.withTransfer(c, "", func(gCtx *gin.Context, transfer *edi.EDITransfer) {
		gCtx.JSON(http.StatusOK, transfer)
	})
}

func (h *Handler) mappingPreview(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	preview, err := h.service.MappingPreview(
		c.Request.Context(),
		repositories.GetEDITransferByIDRequest{
			ID: transferID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, preview)
}

func (h *Handler) approveTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.ApproveTransferRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TransferID = transferID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.service.ApproveTransfer(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, transfer)
}

func (h *Handler) rejectTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.RejectTransferRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TransferID = transferID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	transfer, err := h.service.RejectTransfer(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (h *Handler) cancelTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.service.CancelTransfer(
		c.Request.Context(),
		&ediservice.CancelTransferRequest{
			TransferID: transferID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (h *Handler) expireTransfer(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.service.ExpireTransfer(
		c.Request.Context(),
		&ediservice.ExpireTransferRequest{
			TransferID: transferID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
		actorutil.FromAuthContext(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, transfer)
}

func (h *Handler) listShipmentLinks(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.ShipmentLink], error) {
		return h.service.ListShipmentLinks(
			c.Request.Context(),
			&repositories.ListEDIShipmentLinksRequest{Filter: req},
		)
	})
}

func (h *Handler) getShipmentLink(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	linkID, err := pulid.MustParse(c.Param("linkID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	link, err := h.service.GetShipmentLink(
		c.Request.Context(),
		repositories.GetEDIShipmentLinkByIDRequest{
			ID: linkID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, link)
}

func (h *Handler) listTransferChanges(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	linkID := pulid.Nil
	if rawLinkID := c.Query("shipmentLinkId"); rawLinkID != "" {
		parsed, err := pulid.MustParse(rawLinkID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		linkID = parsed
	}

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.TransferChange], error) {
		return h.service.ListTransferChanges(
			c.Request.Context(),
			&repositories.ListEDITransferChangesRequest{
				Filter:         req,
				ShipmentLinkID: linkID,
			},
		)
	})
}

func (h *Handler) getTransferChange(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	change, err := h.service.GetTransferChange(
		c.Request.Context(),
		repositories.GetEDITransferChangeByIDRequest{
			ID: changeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) applyTransferChange(c *gin.Context) {
	h.transferChangeAction(c, h.service.ApplyTransferChange)
}

func (h *Handler) rejectTransferChange(c *gin.Context) {
	h.transferChangeAction(c, h.service.RejectTransferChange)
}

func (h *Handler) listTenderChanges(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	recipientID := pulid.Nil
	if rawRecipientID := c.Query("recipientId"); rawRecipientID != "" {
		parsed, err := pulid.MustParse(rawRecipientID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		recipientID = parsed
	}
	sourceShipmentID := pulid.Nil
	if rawShipmentID := c.Query("sourceShipmentId"); rawShipmentID != "" {
		parsed, err := pulid.MustParse(rawShipmentID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}
		sourceShipmentID = parsed
	}

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.TenderChange], error) {
		return h.service.ListTenderChanges(
			c.Request.Context(),
			&repositories.ListEDITenderChangesRequest{
				Filter:           req,
				RecipientID:      recipientID,
				SourceShipmentID: sourceShipmentID,
				Status:           edi.TenderChangeStatus(helpers.QueryString(c, "status", "")),
			},
		)
	})
}

func (h *Handler) getTenderChange(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	change, err := h.service.GetTenderChange(
		c.Request.Context(),
		repositories.GetEDITenderChangeByIDRequest{
			ID: changeID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) applyTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.service.ApplyTenderChange)
}

func (h *Handler) rejectTenderChange(c *gin.Context) {
	h.tenderChangeAction(c, h.service.RejectTenderChange)
}

func (h *Handler) tenderChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.TenderChangeActionRequest,
		*services.RequestActor,
	) (*edi.TenderChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.TenderChangeActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ChangeID = changeID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	change, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) transferChangeAction(
	c *gin.Context,
	fn func(
		context.Context,
		*ediservice.TransferChangeActionRequest,
		*services.RequestActor,
	) (*edi.TransferChange, error),
) {
	authCtx := authctx.GetAuthContext(c)
	changeID, err := pulid.MustParse(c.Param("changeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(ediservice.TransferChangeActionRequest)
	if c.Request.ContentLength > 0 {
		if err = c.ShouldBindJSON(req); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}
	req.ChangeID = changeID
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	change, err := fn(c.Request.Context(), req, actorutil.FromAuthContext(authCtx))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, change)
}

func (h *Handler) withTransfer(
	c *gin.Context,
	direction string,
	fn func(*gin.Context, *edi.EDITransfer),
) {
	authCtx := authctx.GetAuthContext(c)
	transferID, err := pulid.MustParse(c.Param("transferID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	transfer, err := h.service.GetTransfer(
		c.Request.Context(),
		repositories.GetEDITransferByIDRequest{
			ID: transferID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			Direction: direction,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	fn(c, transfer)
}
