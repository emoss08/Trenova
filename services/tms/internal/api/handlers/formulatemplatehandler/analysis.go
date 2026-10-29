package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatereviewservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplatestudioservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type backtestRequest struct {
	Source        string `json:"source"`
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
// @Success 200 {object} formulatemplatestudioservice.BacktestResponse
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

	if !h.allowShipmentRead(c, authCtx) {
		return
	}

	response, err := h.studioService.ApprovalImpact(
		c.Request.Context(),
		&formulatemplatestudioservice.ApprovalImpactRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
			Limit:      req.Limit,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, response)
}

// @Summary Compare a formula template's pending content with its last approved snapshot
// @ID formulaTemplateReviewDiff
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} formulatemplatereviewservice.ReviewDiffResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/review-diff [get]
func (h *Handler) reviewDiff(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response, err := h.reviews.ReviewDiff(
		c.Request.Context(),
		&formulatemplatereviewservice.ReviewDiffRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, response)
}

// @Summary Report whether a formula template is ready to submit or approve
// @ID formulaTemplateReadiness
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} formulatemplatereviewservice.ReadinessResponse
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/{templateID}/readiness [get]
func (h *Handler) readiness(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	templateID, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	response, err := h.reviews.Readiness(
		c.Request.Context(),
		&formulatemplatereviewservice.ReadinessRequest{
			TenantInfo: pagination.FromAuthAsUser(authCtx),
			TemplateID: templateID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, response)
}

// @Summary Backtest a formula template candidate against rated shipments
// @ID backtestFormulaTemplate
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Param request body backtestRequest true "Backtest request"
// @Success 200 {object} formulatemplatestudioservice.BacktestResponse
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

	if !h.allowShipmentRead(c, authCtx) {
		return
	}

	response, err := h.studioService.Backtest(
		c.Request.Context(),
		&formulatemplatestudioservice.BacktestRequest{
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
			TemplateID:    templateID,
			Source:        req.Source,
			VersionNumber: req.VersionNumber,
			Limit:         req.Limit,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, response)
}
