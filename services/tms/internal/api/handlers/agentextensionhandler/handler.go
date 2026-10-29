package agentextensionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/extensionportalservice"
	"github.com/emoss08/trenova/internal/modules/agent/app/extensionwebwatchservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *agentextensionservice.Service
	Portals              *extensionportalservice.Service
	WebWatches           *extensionwebwatchservice.Service
	Waits                agentextensionports.WebWatchWaits `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service    *agentextensionservice.Service
	portals    *extensionportalservice.Service
	webWatches *extensionwebwatchservice.Service
	waits      agentextensionports.WebWatchWaits
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		portals:    p.Portals,
		webWatches: p.WebWatches,
		waits:      p.Waits,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceAgentExtension.String()
	api := rg.Group("/agent-extensions")
	api.GET("/catalog/", h.pm.RequirePermission(resource, permission.OpRead), h.catalog)
	api.GET("/:type/config/", h.pm.RequirePermission(resource, permission.OpRead), h.getConfig)
	api.PUT("/:type/config/", h.pm.RequirePermission(resource, permission.OpUpdate), h.updateConfig)
	api.POST("/:type/test/", h.pm.RequirePermission(resource, permission.OpUpdate), h.test)

	portals := rg.Group("/agent-portals")
	portals.GET("/sites/", h.pm.RequirePermission(resource, permission.OpRead), h.listSites)
	portals.POST("/sites/", h.pm.RequirePermission(resource, permission.OpUpdate), h.createSite)
	portals.PUT(
		"/sites/:siteID/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.updateSite,
	)
	portals.DELETE(
		"/sites/:siteID/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.deleteSite,
	)
	portals.GET("/runs/", h.pm.RequirePermission(resource, permission.OpRead), h.listRuns)
	portals.GET(
		"/runs/:runID/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpRead},
			middleware.PermissionCheck{
				Resource:  permission.ResourcePortalAutomation.String(),
				Operation: permission.OpRead,
			},
		),
		h.portalRun,
	)
	portals.GET(
		"/runs/:runID/live/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpUpdate},
			middleware.PermissionCheck{
				Resource:  permission.ResourcePortalAutomation.String(),
				Operation: permission.OpRead,
			},
		),
		h.portalRunLive,
	)
	portals.GET(
		"/runs/:runID/recording/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.runRecording,
	)
	portals.POST(
		"/runs/:runID/sign-in-code/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpUpdate},
			middleware.PermissionCheck{
				Resource:  permission.ResourcePortalAutomation.String(),
				Operation: permission.OpRead,
			},
		),
		h.provideSignInCode,
	)

	watches := rg.Group("/agent-web-watches")
	watches.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.listWebWatches)
	watches.GET(
		"/:watchID/findings/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.webWatchFindings,
	)
	watches.POST(
		"/:watchID/stop/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpUpdate},
			middleware.PermissionCheck{
				Resource:  permission.ResourceWebResearch.String(),
				Operation: permission.OpRead,
			},
		),
		h.stopWebWatch,
	)
}

func tenantOf(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)

	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func typeOf(c *gin.Context) agentextension.Type {
	return agentextension.Type(c.Param("type"))
}

func (h *Handler) catalog(c *gin.Context) {
	result, err := h.service.ListCatalog(c.Request.Context(), tenantOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) getConfig(c *gin.Context) {
	result, err := h.service.GetConfig(c.Request.Context(), tenantOf(c), typeOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) updateConfig(c *gin.Context) {
	req := new(agentextensionports.UpdateAgentExtensionRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)

	result, err := h.service.UpdateConfig(c.Request.Context(), typeOf(c), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}

func (h *Handler) test(c *gin.Context) {
	result, err := h.service.TestConnection(c.Request.Context(), tenantOf(c), typeOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, result)
}
