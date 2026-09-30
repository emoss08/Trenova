package formulatemplatehandler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

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
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		EntityID:   templateID,
		Comment:    req.Comment,
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

// @Summary Request changes on a formula template under review
// @ID requestFormulaTemplateChanges
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Template ID"
// @Param request body approvalActionRequest true "Reviewer comment"
// @Success 200 {object} formulatemplate.FormulaTemplate
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/request-changes [post]
func (h *Handler) requestChanges(c *gin.Context) {
	h.handleApprovalAction(c, h.service.RequestChanges)
}

// @Summary List the review history of a formula template
// @ID listFormulaTemplateReviews
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Template ID"
// @Success 200 {array} formulatemplate.Review
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/reviews [get]
func (h *Handler) listReviews(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	reviews, err := h.service.ListReviews(
		c.Request.Context(),
		&formulatemplateservice.ListReviewsRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, reviews)
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
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
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
