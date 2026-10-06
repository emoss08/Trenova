package assistanthandler

import (
	"net/http"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

type handoffRequest struct {
	AgentDefinitionID string `json:"agentDefinitionId" binding:"required"`
	// Facts are the conversation's pinned facts, carried as the person
	// sees them.
	Facts []string `json:"facts"`
}

// handoff takes the conversation to another agent: a new conversation with
// it, opened with a summary, the pinned facts and the pinned artifacts, and a
// card in this one saying so.
func (h *Handler) handoff(c *gin.Context) {
	if h.handoffs == nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Handing off is not available"))
		return
	}
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body handoffRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	agentID, err := pulid.Parse(body.AgentDefinitionID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := requestActorFromAuthContext(authctx.GetAuthContext(c))
	result, err := h.handoffs.Handoff(c.Request.Context(), &serviceports.HandoffThreadRequest{
		TenantInfo:        req.TenantInfo,
		ThreadID:          req.ID,
		AgentDefinitionID: agentID,
		Facts:             body.Facts,
	}, &actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}
