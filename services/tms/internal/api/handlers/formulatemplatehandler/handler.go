package formulatemplatehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *formulatemplateservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *formulatemplateservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

// TODO: Add permission middleware
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/formula-templates")
	api.GET("/", h.list)
	api.POST("/", h.create)
	api.POST("/bulk-update-status", h.bulkUpdateStatus)
	api.POST("/test", h.testExpression)
	api.POST("/duplicate", h.duplicate)

	idGroup := api.Group("/:templateID")
	idGroup.GET("/", h.get)
	idGroup.PUT("/", h.update)
	idGroup.PATCH("/", h.patch)
	idGroup.GET("/usage", h.getUsage)
	idGroup.GET("/versions", h.listVersions)
	idGroup.GET("/versions/:versionNumber", h.getVersion)
	idGroup.POST("/versions", h.createVersion)
	idGroup.POST("/rollback", h.rollback)
	idGroup.POST("/fork", h.fork)
	idGroup.GET("/compare", h.compareVersions)
	idGroup.GET("/lineage", h.getLineage)
	idGroup.PATCH("/versions/:versionNumber/tags", h.updateVersionTags)

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

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

type testExpressionRequest struct {
	Expression string         `json:"expression"`
	SchemaID   string         `json:"schemaId"`
	Variables  map[string]any `json:"variables"`
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
	var req testExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.SchemaID == "" {
		req.SchemaID = "shipment"
	}

	result := h.service.TestExpression(&formulatemplateservice.TestExpressionRequest{
		Expression: req.Expression,
		SchemaID:   req.SchemaID,
		Variables:  req.Variables,
	})

	c.JSON(http.StatusOK, result)
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
