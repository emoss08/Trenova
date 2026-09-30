package formulatemplatehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

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
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
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
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
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
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
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
			TenantInfo:       pagination.FromAuthAsUser(authCtx),
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
// @Failure 400 {object} helpers.ProblemDetail
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
		h.eh.HandleError(c, errortypes.NewValidationError(
			"from",
			errortypes.ErrRequired,
			"Both 'from' and 'to' version parameters are required and must be positive",
		))
		return
	}

	if fromVersion == toVersion {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"to",
			errortypes.ErrInvalid,
			"The 'from' and 'to' versions must be different",
		))
		return
	}

	diff, err := h.service.CompareVersions(
		c.Request.Context(),
		&repositories.CompareVersionsRequest{
			TenantInfo:  pagination.FromAuthAsUser(authCtx),
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
			TenantInfo: pagination.FromAuthAsUser(authCtx),
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
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
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
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, versions)
}
