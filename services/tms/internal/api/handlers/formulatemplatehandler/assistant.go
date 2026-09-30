package formulatemplatehandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type openAssistantThreadRequest struct {
	TemplateID string `json:"templateId"`
}

// @Summary Open the formula assistant's conversation
// @Description Opens, or returns, the caller's conversation with the formula assistant about one formula template, or a new one for a template not yet saved. Turns are then asked on it through the assistant's turn routes, carrying the editor's draft. Requires formula template read, assistant create, and access to the formula assistant.
// @ID openFormulaAssistantThread
// @Tags Formula Templates
// @Accept json
// @Produce json
// @Param request body openAssistantThreadRequest true "The template the conversation is about, when it is saved"
// @Success 200 {object} services.PageThread
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-templates/ai/thread/ [post]
func (h *Handler) openAssistantThread(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req openAssistantThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"body", errortypes.ErrInvalid, "Invalid request body",
		))
		return
	}

	templateID := pulid.Nil
	if raw := strings.TrimSpace(req.TemplateID); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"templateId", errortypes.ErrInvalid, "Invalid formula template ID",
			))
			return
		}
		templateID = parsed
	}

	if h.pageAssistant == nil {
		h.eh.HandleError(c, errortypes.NewBusinessError(
			"The formula assistant is not available on this server",
		))
		return
	}

	tenant := pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	if templateID.IsNotNil() {
		if _, err := h.service.GetByID(c.Request.Context(), repositories.GetFormulaTemplateByIDRequest{
			TemplateID: templateID,
			TenantInfo: tenant,
		}); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}

	actor := services.RequestActor{
		PrincipalType:  services.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
	opened, err := h.pageAssistant.OpenPageThread(
		c.Request.Context(),
		&services.OpenPageThreadRequest{
			TenantInfo:  tenant,
			Origin:      conversation.ThreadOriginFormula,
			SubjectType: agent.SubjectFormulaTemplate,
			SubjectID:   templateID,
		},
		&actor,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, opened)
}
