package formulatemplatehandler

import (
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *formulatemplateservice.Service
	PageAssistant        services.PageAssistant `optional:"true"`
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	PermissionEngine     services.PermissionEngine
}

type Handler struct {
	service       *formulatemplateservice.Service
	pageAssistant services.PageAssistant
	eh            *helpers.ErrorHandler
	pm            *middleware.PermissionMiddleware
	permEngine    services.PermissionEngine
}

func New(p Params) *Handler {
	return &Handler{
		service:       p.Service,
		pageAssistant: p.PageAssistant,
		eh:            p.ErrorHandler,
		pm:            p.PermissionMiddleware,
		permEngine:    p.PermissionEngine,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceFormulaTemplate.String()
	requireRead := h.pm.RequirePermission(resource, permission.OpRead)
	requireCreate := h.pm.RequirePermission(resource, permission.OpCreate)
	requireUpdate := h.pm.RequirePermission(resource, permission.OpUpdate)
	requireDuplicate := h.pm.RequirePermission(resource, permission.OpDuplicate)
	requireSubmit := h.pm.RequirePermission(resource, permission.OpSubmit)
	requireApprove := h.pm.RequirePermission(resource, permission.OpApprove)
	requireReject := h.pm.RequirePermission(resource, permission.OpReject)
	requireAuthoring := h.pm.RequireAnyPermission(
		middleware.PermissionCheck{Resource: resource, Operation: permission.OpCreate},
		middleware.PermissionCheck{Resource: resource, Operation: permission.OpUpdate},
	)

	api := rg.Group("/formula-templates")
	api.GET("/", requireRead, h.list)
	api.POST("/", requireCreate, h.create)
	api.GET("/schema", requireRead, h.getSchema)
	api.POST("/bulk-update-status", requireUpdate, h.bulkUpdateStatus)
	api.POST("/test", requireAuthoring, h.testExpression)
	api.POST("/duplicate", requireDuplicate, h.duplicate)
	api.POST("/import", requireCreate, h.importTemplates)
	api.GET("/standards", requireRead, h.listStandards)
	api.POST("/install-standards", requireCreate, h.installStandards)
	api.POST(
		"/ai/thread/",
		requireRead,
		h.pm.RequirePermission(permission.ResourceAssistant.String(), permission.OpCreate),
		h.openAssistantThread,
	)

	idGroup := api.Group("/:templateID")
	idGroup.GET("/", requireRead, h.get)
	idGroup.PUT("/", requireUpdate, h.update)
	idGroup.PATCH("/", requireUpdate, h.patch)
	idGroup.GET("/usage", requireRead, h.getUsage)
	idGroup.GET("/versions", requireRead, h.listVersions)
	idGroup.GET("/versions/:versionNumber", requireRead, h.getVersion)
	idGroup.POST("/versions", requireUpdate, h.createVersion)
	idGroup.POST("/rollback", requireUpdate, h.rollback)
	idGroup.POST("/fork", requireCreate, h.fork)
	idGroup.GET("/compare", requireRead, h.compareVersions)
	idGroup.GET("/lineage", requireRead, h.getLineage)
	idGroup.PATCH("/versions/:versionNumber/tags", requireUpdate, h.updateVersionTags)
	idGroup.POST("/submit", requireSubmit, h.submit)
	idGroup.POST("/approve", requireApprove, h.approve)
	idGroup.POST("/reject", requireReject, h.reject)
	idGroup.POST("/request-changes", requireReject, h.requestChanges)
	idGroup.GET("/reviews", requireRead, h.listReviews)
	idGroup.POST("/backtest", requireAuthoring, h.backtest)
	idGroup.POST("/impact", requireRead, h.approvalImpact)
	idGroup.GET("/readiness", requireRead, h.readiness)
	idGroup.GET("/review-diff", requireRead, h.reviewDiff)
	idGroup.GET("/test-cases", requireRead, h.listTestCases)
	idGroup.POST("/test-cases", requireUpdate, h.createTestCase)
	idGroup.PUT("/test-cases/:testCaseID", requireUpdate, h.updateTestCase)
	idGroup.DELETE("/test-cases/:testCaseID", requireUpdate, h.deleteTestCase)
	idGroup.POST("/test-cases/run", requireAuthoring, h.runTestCases)
	idGroup.GET("/versions/scheduled", requireRead, h.listScheduledVersions)
	idGroup.PATCH(
		"/versions/:versionNumber/effective-date",
		requireApprove,
		h.updateVersionEffectiveDate,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", requireRead, h.selectOptions)
	selectOptions.GET("/:templateID", requireRead, h.getOption)
}
