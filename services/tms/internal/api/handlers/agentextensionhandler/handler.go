package agentextensionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
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
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *agentextensionservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
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

	analyses := rg.Group("/agent-analysis")
	analyses.GET("/runs/", h.pm.RequirePermission(resource, permission.OpRead), h.listAnalysisRuns)
	analyses.GET(
		"/runs/:runID/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpRead},
			middleware.PermissionCheck{
				Resource:  permission.ResourceDataAnalysis.String(),
				Operation: permission.OpRead,
			},
		),
		h.analysisRun,
	)
	analyses.GET(
		"/runs/:runID/charts/:position/",
		h.pm.RequireAnyPermission(
			middleware.PermissionCheck{Resource: resource, Operation: permission.OpRead},
			middleware.PermissionCheck{
				Resource:  permission.ResourceDataAnalysis.String(),
				Operation: permission.OpRead,
			},
		),
		h.analysisChart,
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getConfig(c *gin.Context) {
	result, err := h.service.GetConfig(c.Request.Context(), tenantOf(c), typeOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) test(c *gin.Context) {
	result, err := h.service.TestConnection(c.Request.Context(), tenantOf(c), typeOf(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
