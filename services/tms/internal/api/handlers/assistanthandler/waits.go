package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// A conversation's waits are its agent's parked work, read and cancelled by
// the conversation's owner as part of arranging it.
func (h *Handler) registerWaitRoutes(api *gin.RouterGroup, resource string) {
	read := h.pm.RequirePermission(resource, permission.OpRead)

	api.GET("/threads/:threadID/waits/", read, h.listWaits)
	api.POST("/threads/:threadID/waits/:waitID/cancel/", read, h.cancelWait)
}

type waitListResponse struct {
	Items []*agentwait.Wait `json:"items"`
}

func (h *Handler) listWaits(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.waits.List(c.Request.Context(), &repositories.ListThreadWaitsRequest{
		ThreadID:   req.ID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, waitListResponse{Items: items})
}

func (h *Handler) cancelWait(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	waitID, err := pulid.Parse(c.Param("waitID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	wait, err := h.waits.Cancel(c.Request.Context(), &serviceports.CancelWaitRequest{
		ID:         waitID,
		TenantInfo: req.TenantInfo,
		ThreadID:   req.ID,
		UserID:     authctx.GetAuthContext(c).UserID,
		By:         "you",
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, wait)
}
