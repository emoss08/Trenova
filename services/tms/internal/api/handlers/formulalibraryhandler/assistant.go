package formulalibraryhandler

import (
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type openAssistantThreadRequest struct {
	LibraryID string `json:"libraryId"`
}

// @Summary Open the formula assistant's conversation about a library
// @Description Opens, or returns, the caller's conversation with the formula assistant about one formula library, or a new one for a library not yet saved. Turns are then asked on it through the assistant's turn routes, carrying the library editor's draft. Requires formula library read, assistant create, and access to the formula assistant.
// @ID openFormulaLibraryAssistantThread
// @Tags Formula Libraries
// @Accept json
// @Produce json
// @Param request body openAssistantThreadRequest true "The library the conversation is about, when it is saved"
// @Success 200 {object} services.PageThread
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /formula-libraries/ai/thread/ [post]
func (h *Handler) openAssistantThread(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	var req openAssistantThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"body", errortypes.ErrInvalid, "Invalid request body",
		))
		return
	}

	libraryID := pulid.Nil
	if raw := strings.TrimSpace(req.LibraryID); raw != "" {
		parsed, err := pulid.Parse(raw)
		if err != nil {
			h.eh.HandleError(c, errortypes.NewValidationError(
				"libraryId", errortypes.ErrInvalid, "Invalid formula library ID",
			))
			return
		}
		libraryID = parsed
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
	if libraryID.IsNotNil() {
		if _, err := h.service.GetByID(
			c.Request.Context(),
			&formulalibraryports.GetLibraryByIDRequest{
				LibraryID:  libraryID,
				TenantInfo: tenant,
			},
		); err != nil {
			h.eh.HandleError(c, err)
			return
		}
	}

	opened, err := h.pageAssistant.OpenPageThread(
		c.Request.Context(),
		&assistantports.OpenPageThreadRequest{
			TenantInfo:  tenant,
			Origin:      conversation.ThreadOriginFormula,
			SubjectType: agent.SubjectFormulaLibrary,
			SubjectID:   libraryID,
		},
		requestActor(authCtx),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, opened)
}

func requestActor(authCtx *authctx.AuthContext) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  kernelports.PrincipalType(authCtx.PrincipalType),
		PrincipalID:    authCtx.PrincipalID,
		UserID:         authCtx.UserID,
		APIKeyID:       authCtx.APIKeyID,
		BusinessUnitID: authCtx.BusinessUnitID,
		OrganizationID: authCtx.OrganizationID,
	}
}
