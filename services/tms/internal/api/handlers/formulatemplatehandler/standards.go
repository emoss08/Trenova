package formulatemplatehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// @Summary List the standard formula template catalog
// @ID listStandardFormulaTemplates
// @Tags Formula Templates
// @Produce json
// @Success 200 {array} formulatemplateimportservice.StandardTemplate
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/standards [get]
func (h *Handler) listStandards(c *gin.Context) {
	standards, err := h.imports.ListStandards()
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, standards)
}

// @Summary Get the inputs a rating method prices a shipment with
// @ID getPricingTemplateInputs
// @Tags Formula Templates
// @Produce json
// @Param templateID path string true "Formula template ID"
// @Success 200 {object} formulatemplateservice.PricingInputs
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /pricing-templates/{templateID}/inputs [get]
func (h *Handler) pricingInputs(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	mayReadCustom, err := h.pm.Allowed(
		c,
		permission.ResourceFormulaTemplate.String(),
		permission.OpRead,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	inputs, err := h.service.PricingInputs(
		c.Request.Context(),
		&formulatemplateservice.PricingInputsRequest{
			TenantInfo:    pagination.FromAuthAsUser(authCtx),
			TemplateID:    id,
			MayReadCustom: mayReadCustom,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, inputs)
}
