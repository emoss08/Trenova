package formulatemplatehandler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/formulaassistantservice"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *formulatemplateservice.Service
	AssistantService     *formulaassistantservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
	PermissionEngine     services.PermissionEngine
}

type Handler struct {
	service    *formulatemplateservice.Service
	assistant  *formulaassistantservice.Service
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
	permEngine services.PermissionEngine
}

func New(p Params) *Handler {
	return &Handler{
		service:    p.Service,
		assistant:  p.AssistantService,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
		permEngine: p.PermissionEngine,
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
	api.POST("/install-standards", requireCreate, h.installStandards)
	api.POST("/ai/generate", requireAuthoring, h.aiGenerate)
	api.POST("/ai/explain", requireRead, h.aiExplain)

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
	idGroup.POST("/backtest", requireAuthoring, h.backtest)
	idGroup.POST("/impact", requireRead, h.approvalImpact)
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
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:templateID", h.getOption)
}

// @Summary List formula templates
// @ID listFormulaTemplates
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param type query string false "Filter by template type"
// @Param status query string false "Filter by template status"
// @Success 200 {object} pagination.Response[[]formulatemplate.FormulaTemplate]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			return h.service.List(c.Request.Context(), &repositories.ListFormulaTemplatesRequest{
				Filter: req,
				Type:   helpers.QueryString(c, "type"),
				Status: helpers.QueryString(c, "status"),
			})
		},
	)
}

// @Summary Get a formula template
// @ID getFormulaTemplate
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetFormulaTemplateByIDRequest{
			TemplateID: id,
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

	c.JSON(http.StatusOK, entity)
}

// @Summary Get a formula template option
// @ID getFormulaTemplateOption
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/select-options/{templateID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetFormulaTemplateByIDRequest{
			TemplateID: templateID,
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

// @Summary List formula template options
// @ID listFormulaTemplateOptions
// @Tags Formula Templates
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]formulatemplate.FormulaTemplate]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*formulatemplate.FormulaTemplate], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.FormulaTemplateSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get formula template usage
// @ID getFormulaTemplateUsage
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} repositories.GetTemplateUsageResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/usage [get]
func (h *Handler) getUsage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	usage, err := h.service.GetUsage(
		c.Request.Context(),
		&repositories.GetTemplateUsageRequest{
			TemplateID: id,
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

	c.JSON(http.StatusOK, usage)
}

// @Summary Create a formula template
// @ID createFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulatemplate.FormulaTemplate true "Formula template payload"
// @Success 201 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(formulatemplate.FormulaTemplate)
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	createdEntity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, createdEntity)
}

// @Summary Update a formula template
// @ID updateFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplate.FormulaTemplate true "Formula template payload"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(formulatemplate.FormulaTemplate)
	entity.ID = templateID
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Duplicate formula templates
// @ID duplicateFormulaTemplates
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body repositories.BulkDuplicateFormulaTemplateRequest true "Bulk duplicate request"
// @Success 200 {array} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/duplicate [post]
func (h *Handler) duplicate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.BulkDuplicateFormulaTemplateRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	entity, err := h.service.Duplicate(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Bulk update formula template statuses
// @ID bulkUpdateFormulaTemplateStatus
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateFormulaTemplateStatusRequest true "Bulk status update request"
// @Success 200 {array} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/bulk-update-status [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.BulkUpdateFormulaTemplateStatusRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	results, err := h.service.BulkUpdateStatus(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, results)
}

// @Summary Patch a formula template
// @ID patchFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplate.FormulaTemplate true "Formula template payload"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetFormulaTemplateByIDRequest{
			TemplateID: templateID,
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

	// A partial update never moves status; that belongs to the review
	// workflow, so whatever the body says about it is dropped here.
	status := existing.Status
	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	existing.Status = status

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Describe the variables and functions available to formula expressions
// @ID getFormulaSchema
// @Tags Formula Templates
// @Produce json
// @Param schemaId query string false "Formula schema identifier" default(shipment)
// @Success 200 {object} formulatemplatetypes.SchemaDescription
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/schema [get]
func (h *Handler) getSchema(c *gin.Context) {
	schemaID := helpers.QueryString(c, "schemaId")
	if schemaID == "" {
		schemaID = "shipment"
	}

	var description *formulatemplatetypes.SchemaDescription
	description, err := h.service.DescribeSchema(schemaID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, description)
}

// @Summary Import formula templates from an exported JSON payload
// @ID importFormulaTemplates
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulatemplateservice.ImportTemplatesRequest true "Import request"
// @Success 200 {object} formulatemplateservice.ImportTemplatesResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/import [post]
func (h *Handler) importTemplates(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req formulatemplateservice.ImportTemplatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	result, err := h.service.Import(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary List a formula template's saved test scenarios
// @ID listFormulaTemplateTestCases
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {array} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases [get]
func (h *Handler) listTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	cases, err := h.service.ListTestCases(c.Request.Context(), repositories.ListTestCasesRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		TemplateID: templateID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, cases)
}

// @Summary Create a test scenario for a formula template
// @ID createFormulaTemplateTestCase
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplateservice.TestCaseInput true "Test scenario"
// @Success 200 {object} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases [post]
func (h *Handler) createTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var input formulatemplateservice.TestCaseInput
	if err = c.ShouldBindJSON(&input); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.CreateTestCase(
		c.Request.Context(),
		&formulatemplateservice.CreateTestCaseRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			TestCaseInput: input,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, created)
}

// @Summary Update a formula template test scenario
// @ID updateFormulaTemplateTestCase
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param testCaseID path string true "Test scenario ID"
// @Param request body formulatemplateservice.UpdateTestCaseRequest true "Test scenario"
// @Success 200 {object} formulatemplate.TestCase
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/{testCaseID} [put]
func (h *Handler) updateTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req formulatemplateservice.UpdateTestCaseRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.TemplateID = templateID
	req.TestCaseID = testCaseID

	updated, err := h.service.UpdateTestCase(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a formula template test scenario
// @ID deleteFormulaTemplateTestCase
// @Tags Formula Templates
// @Param templateID path string true "Formula template ID"
// @Param testCaseID path string true "Test scenario ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/{testCaseID} [delete]
func (h *Handler) deleteTestCase(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	testCaseID, err := pulid.MustParse(c.Param("testCaseID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	err = h.service.DeleteTestCase(c.Request.Context(), repositories.GetTestCaseByIDRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		TemplateID: templateID,
		TestCaseID: testCaseID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Run a formula template's test scenarios
// @ID runFormulaTemplateTestCases
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body formulatemplateservice.RunTestCasesRequest true "Optional candidate content"
// @Success 200 {object} formulatemplateservice.RunTestCasesResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/test-cases/run [post]
func (h *Handler) runTestCases(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req formulatemplateservice.RunTestCasesRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	req.TemplateID = templateID

	result, err := h.service.RunTestCases(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Install the standard formula template library for this organization
// @ID installStandardFormulaTemplates
// @Tags Formula Templates
// @Produce json
// @Success 200 {object} formulatemplateservice.InstallStandardsResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/install-standards [post]
func (h *Handler) installStandards(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	result, err := h.service.InstallStandards(c.Request.Context(), pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Generate a formula expression from a natural-language description
// @ID generateFormulaExpression
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulaassistantservice.GenerateFormulaRequest true "Generation request"
// @Success 200 {object} formulaassistantservice.GenerateFormulaResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/generate [post]
func (h *Handler) aiGenerate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req formulaassistantservice.GenerateFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	result, err := h.assistant.GenerateFormula(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Explain a formula expression in plain English
// @ID explainFormulaExpression
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body formulaassistantservice.ExplainFormulaRequest true "Explanation request"
// @Success 200 {object} formulaassistantservice.ExplainFormulaResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/explain [post]
func (h *Handler) aiExplain(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req formulaassistantservice.ExplainFormulaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	result, err := h.assistant.ExplainFormula(c.Request.Context(), &req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

type testExpressionRequest struct {
	Expression string                              `json:"expression"`
	SchemaID   string                              `json:"schemaId"`
	Variables  map[string]any                      `json:"variables"`
	ShipmentID string                              `json:"shipmentId"`
	Breakdowns []*formulatypes.BreakdownDefinition `json:"breakdowns"`
	MinCharge  *string                             `json:"minCharge"`
	MaxCharge  *string                             `json:"maxCharge"`
	// RoundingMode and RoundingPrecision are the charge policy under test;
	// omitted means the default the template would store.
	RoundingMode      string `json:"roundingMode"`
	RoundingPrecision *int32 `json:"roundingPrecision"`
}

// @Summary Test a formula expression
// @ID testFormulaExpression
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body testExpressionRequest true "Expression test request"
// @Success 200 {object} formulatemplateservice.TestExpressionResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/test [post]
func (h *Handler) testExpression(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req testExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.SchemaID == "" {
		req.SchemaID = "shipment"
	}

	serviceReq := &formulatemplateservice.TestExpressionRequest{
		Expression: req.Expression,
		SchemaID:   req.SchemaID,
		Variables:  req.Variables,
		Breakdowns: req.Breakdowns,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}

	minCharge, err := parseGuardrailCharge("minCharge", req.MinCharge)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	maxCharge, err := parseGuardrailCharge("maxCharge", req.MaxCharge)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if minCharge.Valid && maxCharge.Valid &&
		minCharge.Decimal.GreaterThan(maxCharge.Decimal) {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"minCharge",
			errortypes.ErrInvalid,
			"Minimum charge cannot exceed maximum charge",
		))
		return
	}
	serviceReq.MinCharge = minCharge
	serviceReq.MaxCharge = maxCharge

	policy, err := parseRoundingPolicy(req.RoundingMode, req.RoundingPrecision)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	serviceReq.RoundingMode = policy.RoundingMode
	serviceReq.RoundingPrecision = policy.RoundingPrecision

	if req.ShipmentID != "" {
		shipmentID, err := pulid.MustParse(req.ShipmentID)
		if err != nil {
			h.eh.HandleError(c, err)
			return
		}

		if !h.allowShipmentRead(c, authCtx) {
			return
		}

		serviceReq.ShipmentID = &shipmentID
	}

	result := h.service.TestExpression(c.Request.Context(), serviceReq)

	c.JSON(http.StatusOK, result)
}

func (h *Handler) allowShipmentRead(c *gin.Context, authCtx *authctx.AuthContext) bool {
	result, err := h.permEngine.Check(
		c.Request.Context(),
		middleware.BuildPermissionCheckRequest(
			authCtx,
			permission.ResourceShipment.String(),
			permission.OpRead,
		),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return false
	}

	if !result.Allowed {
		h.eh.HandleError(c, errortypes.NewAuthorizationError(
			"You don't have permission to read shipments",
		))
		return false
	}

	return true
}

func parseRoundingPolicy(mode string, precision *int32) (formulatypes.ChargePolicy, error) {
	policy := formulatypes.ChargePolicy{RoundingMode: ratetypes.RoundingMode(mode)}

	if mode != "" && !policy.RoundingMode.IsValid() {
		return policy, errortypes.NewValidationError(
			"roundingMode",
			errortypes.ErrInvalid,
			"Must be one of HalfUp, HalfEven, Up, Down, or None",
		)
	}

	if precision != nil {
		if *precision < 0 || *precision > formulatypes.MaxRoundingPrecision {
			return policy, errortypes.NewValidationError(
				"roundingPrecision",
				errortypes.ErrInvalid,
				"Must be between 0 and 4",
			)
		}
		policy.RoundingPrecision = *precision
	} else if mode != "" {
		policy.RoundingPrecision = formulatypes.DefaultRoundingPrecision
	}

	return policy, nil
}

func parseGuardrailCharge(field string, raw *string) (decimal.NullDecimal, error) {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}, nil
	}

	value, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Must be a valid decimal number",
		)
	}

	if value.IsNegative() {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			field,
			errortypes.ErrInvalid,
			"Cannot be negative",
		)
	}

	return decimal.NullDecimal{Decimal: value, Valid: true}, nil
}

// @Summary List formula template versions
// @ID listFormulaTemplateVersions
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]formulatemplate.FormulaTemplateVersion]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions [get]
func (h *Handler) listVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*formulatemplate.FormulaTemplateVersion], error) {
			return h.service.ListVersions(c.Request.Context(), &repositories.ListVersionsRequest{
				Filter:     req,
				TemplateID: templateID,
			})
		},
	)
}

// @Summary Get a formula template version
// @ID getFormulaTemplateVersion
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param versionNumber path int true "Version number"
// @Success 200 {object} formulatemplate.FormulaTemplateVersion
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions/{versionNumber} [get]
func (h *Handler) getVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versionNumberStr := c.Param("versionNumber")
	versionNumber, err := strconv.ParseInt(versionNumberStr, 10, 64)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	version, err := h.service.GetVersion(
		c.Request.Context(),
		&repositories.GetVersionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			VersionNumber: versionNumber,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

type createVersionRequest struct {
	ChangeMessage string `json:"changeMessage"`
}

// @Summary Create a formula template version
// @ID createFormulaTemplateVersion
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body createVersionRequest true "Create version request"
// @Success 201 {object} formulatemplate.FormulaTemplateVersion
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions [post]
func (h *Handler) createVersion(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req createVersionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	version, err := h.service.CreateVersion(
		c.Request.Context(),
		&repositories.CreateVersionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			ChangeMessage: req.ChangeMessage,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, version)
}

type rollbackRequest struct {
	TargetVersion int64  `json:"targetVersion"`
	ChangeMessage string `json:"changeMessage"`
}

// @Summary Roll back a formula template
// @ID rollbackFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body rollbackRequest true "Rollback request"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/rollback [post]
func (h *Handler) rollback(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req rollbackRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	template, err := h.service.Rollback(
		c.Request.Context(),
		&repositories.RollbackRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			TargetVersion: req.TargetVersion,
			ChangeMessage: req.ChangeMessage,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}

type forkRequest struct {
	NewName       string `json:"newName"`
	SourceVersion *int64 `json:"sourceVersion"`
	ChangeMessage string `json:"changeMessage"`
}

// @Summary Fork a formula template
// @ID forkFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body forkRequest true "Fork request"
// @Success 201 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/fork [post]
func (h *Handler) fork(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req forkRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	template, err := h.service.Fork(
		c.Request.Context(),
		&repositories.ForkTemplateRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			SourceTemplateID: templateID,
			SourceVersion:    req.SourceVersion,
			NewName:          req.NewName,
			ChangeMessage:    req.ChangeMessage,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, template)
}

// @Summary Compare formula template versions
// @ID compareFormulaTemplateVersions
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param from query int true "From version"
// @Param to query int true "To version"
// @Success 200 {object} gin.H
// @Failure 400 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/compare [get]
func (h *Handler) compareVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	fromVersion := helpers.QueryInt64(c, "from", 0)
	toVersion := helpers.QueryInt64(c, "to", 0)

	if fromVersion <= 0 || toVersion <= 0 {
		c.JSON(
			http.StatusBadRequest,
			gin.H{
				"error": "Both 'from' and 'to' version parameters are required and must be positive",
			},
		)
		return
	}

	if fromVersion == toVersion {
		c.JSON(
			http.StatusBadRequest,
			gin.H{"error": "The 'from' and 'to' versions must be different"},
		)
		return
	}

	diff, err := h.service.CompareVersions(
		c.Request.Context(),
		&repositories.CompareVersionsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:  templateID,
			FromVersion: fromVersion,
			ToVersion:   toVersion,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, diff)
}

// @Summary Get formula template lineage
// @ID getFormulaTemplateLineage
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} gin.H
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/lineage [get]
func (h *Handler) getLineage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	lineage, err := h.service.GetLineage(
		c.Request.Context(),
		&repositories.GetLineageRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, lineage)
}

type updateVersionTagsRequest struct {
	Tags []string `json:"tags"`
}

// @Summary Update formula template version tags
// @ID updateFormulaTemplateVersionTags
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param versionNumber path int true "Version number"
// @Param request body updateVersionTagsRequest true "Version tag update request"
// @Success 200 {object} formulatemplate.FormulaTemplateVersion
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions/{versionNumber}/tags [patch]
func (h *Handler) updateVersionTags(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versionNumberStr := c.Param("versionNumber")
	versionNumber, err := strconv.ParseInt(versionNumberStr, 10, 64)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req updateVersionTagsRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	version, err := h.service.UpdateVersionTags(
		c.Request.Context(),
		&repositories.UpdateVersionTagsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			VersionNumber: versionNumber,
			Tags:          req.Tags,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

type approvalActionRequest struct {
	Comment string `json:"comment"`
}

func (h *Handler) handleApprovalAction(
	c *gin.Context,
	action func(
		ctx context.Context,
		req *formulatemplateservice.ApprovalActionRequest,
	) (*formulatemplate.FormulaTemplate, error),
) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req approvalActionRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	template, err := action(c.Request.Context(), &formulatemplateservice.ApprovalActionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		EntityID: templateID,
		Comment:  req.Comment,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, template)
}

// @Summary Submit a formula template for review
// @ID submitFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body approvalActionRequest true "Submit request"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/submit [post]
func (h *Handler) submit(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Submit)
}

// @Summary Approve a formula template
// @ID approveFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body approvalActionRequest true "Approve request"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/approve [post]
func (h *Handler) approve(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Approve)
}

// @Summary Reject a formula template
// @ID rejectFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body approvalActionRequest true "Reject request"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/reject [post]
func (h *Handler) reject(c *gin.Context) {
	h.handleApprovalAction(c, h.service.Reject)
}

type updateEffectiveDateRequest struct {
	EffectiveFrom *int64 `json:"effectiveFrom"`
}

// @Summary Update a formula template version effective date
// @ID updateFormulaTemplateVersionEffectiveDate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param versionNumber path int true "Version number"
// @Param request body updateEffectiveDateRequest true "Effective date update request"
// @Success 200 {object} formulatemplate.FormulaTemplateVersion
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions/{versionNumber}/effective-date [patch]
func (h *Handler) updateVersionEffectiveDate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versionNumber, err := strconv.ParseInt(c.Param("versionNumber"), 10, 64)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req updateEffectiveDateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	version, err := h.service.UpdateVersionEffectiveDate(
		c.Request.Context(),
		&repositories.UpdateEffectiveDateRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			VersionNumber: versionNumber,
			EffectiveFrom: req.EffectiveFrom,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, version)
}

// @Summary List scheduled formula template versions
// @ID listScheduledFormulaTemplateVersions
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {array} formulatemplate.FormulaTemplateVersion
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/versions/scheduled [get]
func (h *Handler) listScheduledVersions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versions, err := h.service.ListScheduledVersions(
		c.Request.Context(),
		&repositories.ListScheduledVersionsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}

type backtestRequest struct {
	Expression    string `json:"expression"`
	VersionNumber *int64 `json:"versionNumber"`
	Limit         int    `json:"limit"`
}

type approvalImpactRequest struct {
	Limit int `json:"limit"`
}

// @Summary Compare a template's pending content against the versions its shipments actually priced with
// @ID formulaTemplateApprovalImpact
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body approvalImpactRequest true "Impact request"
// @Success 200 {object} formulatemplateservice.BacktestResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/impact [post]
func (h *Handler) approvalImpact(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req approvalImpactRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response, err := h.service.ApprovalImpact(
		c.Request.Context(),
		&formulatemplateservice.ApprovalImpactRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID: templateID,
			Limit:      req.Limit,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Backtest a formula template candidate against rated shipments
// @ID backtestFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body backtestRequest true "Backtest request"
// @Success 200 {object} formulatemplateservice.BacktestResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/backtest [post]
func (h *Handler) backtest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var req backtestRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response, err := h.service.Backtest(
		c.Request.Context(),
		&formulatemplateservice.BacktestRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			TemplateID:    templateID,
			Expression:    req.Expression,
			VersionNumber: req.VersionNumber,
			Limit:         req.Limit,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
