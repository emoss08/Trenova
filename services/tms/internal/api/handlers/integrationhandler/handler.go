package integrationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/samsarasyncservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/samsarajobs"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	IntegrationService   *integrationservice.Service
	SyncService          *samsarasyncservice.Service
	JobsManager          *samsarajobs.Manager
	PermissionMiddleware *middleware.PermissionMiddleware
	ErrorHandler         *helpers.ErrorHandler
}

type Handler struct {
	integrationService *integrationservice.Service
	syncService        *samsarasyncservice.Service
	jobsManager        *samsarajobs.Manager
	pm                 *middleware.PermissionMiddleware
	eh                 *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		integrationService: p.IntegrationService,
		syncService:        p.SyncService,
		jobsManager:        p.JobsManager,
		pm:                 p.PermissionMiddleware,
		eh:                 p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/integrations")
	api.GET(
		"/catalog/",
		h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
		h.listCatalog,
	)

	api.GET(
		"/:type/config/",
		h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
		h.getConfig,
	)
	api.PUT(
		"/:type/config/",
		h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpUpdate),
		h.updateConfig,
	)
	api.POST(
		"/:type/test-connection/",
		h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpUpdate),
		h.testConnection,
	)
	api.GET(
		"/:type/runtime-config/",
		h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
		h.getRuntimeConfig,
	)

	samsaraAPI := api.Group("/samsara")
	{
		workerAPI := samsaraAPI.Group("/workers")
		{ //nolint:gocritic // this is mainly for code organization
			workerAPI.GET(
				"/sync/readiness/",
				h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
				h.getWorkerSyncReadiness,
			)
			workerAPI.GET(
				"/sync/drift/",
				h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
				h.getWorkerSyncDrift,
			)
			workerAPI.POST(
				"/sync/drift/detect/",
				h.pm.RequirePermission(
					permission.ResourceIntegration.String(),
					permission.OpUpdate,
				),
				h.detectWorkerSyncDrift,
			)
			workerAPI.POST(
				"/sync/drift/repair/",
				h.pm.RequirePermission(
					permission.ResourceIntegration.String(),
					permission.OpUpdate,
				),
				h.repairWorkerSyncDrift,
			)
			workerAPI.POST(
				"/sync/",
				h.pm.RequirePermission(
					permission.ResourceIntegration.String(),
					permission.OpUpdate,
				),
				h.startWorkerSync,
			)
			workerAPI.GET(
				"/sync/:workflowID/",
				h.pm.RequirePermission(permission.ResourceIntegration.String(), permission.OpRead),
				h.getWorkerSyncStatus,
			)
		}
	}
}

func (h *Handler) listCatalog(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.integrationService.ListCatalog(
		c.Request.Context(),
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getConfig(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	typ := integration.Type(c.Param("type"))

	result, err := h.integrationService.GetConfig(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		typ,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) updateConfig(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	typ := integration.Type(c.Param("type"))

	req := new(services.UpdateConfigRequest)
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.integrationService.UpdateConfig(
		c.Request.Context(),
		req.TenantInfo,
		typ,
		req,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getRuntimeConfig(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	typ := integration.Type(c.Param("type"))

	result, err := h.integrationService.GetRuntimeConfig(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		typ,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result.Config)
}

func (h *Handler) testConnection(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	typ := integration.Type(c.Param("type"))

	result, err := h.integrationService.TestConnection(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		typ,
		authCtx.UserID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) startWorkerSync(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.jobsManager.StartWorkersSyncWorkflow(
		c.Request.Context(),
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

	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) getWorkerSyncStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	workflowID := c.Param("workflowID")
	runID := helpers.QueryStringTrimmed(c, "runId")
	result, err := h.jobsManager.GetWorkersSyncWorkflowStatus(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		workflowID,
		runID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getWorkerSyncDrift(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.syncService.GetWorkerSyncDrift(
		c.Request.Context(),
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) detectWorkerSyncDrift(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.syncService.DetectWorkerSyncDrift(
		c.Request.Context(),
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) repairWorkerSyncDrift(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := services.RepairWorkerSyncDriftRequest{}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.syncService.RepairWorkerSyncDrift(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		req,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getWorkerSyncReadiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.syncService.GetWorkerSyncReadiness(
		c.Request.Context(),
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

	c.JSON(http.StatusOK, result)
}
