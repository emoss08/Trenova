package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

// @Summary List the standard formula template catalog
// @ID listStandardFormulaTemplates
// @Tags Formula Templates
// @Produce json
// @Success 200 {array} formulatemplateservice.StandardTemplate
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/standards [get]
func (h *Handler) listStandards(c *gin.Context) {
	standards, err := h.service.ListStandards()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, standards)
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
