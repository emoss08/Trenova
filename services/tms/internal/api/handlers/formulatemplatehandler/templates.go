package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

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
			TenantInfo: pagination.FromAuthAsUser(authCtx),
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
			TenantInfo: pagination.FromAuth(authCtx),
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
			TenantInfo: pagination.FromAuthAsUser(authCtx),
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
			TenantInfo: pagination.FromAuthAsUser(authCtx),
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
