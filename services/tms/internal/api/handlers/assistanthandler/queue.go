package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/assistantqueueservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerQueueRoutes(api *gin.RouterGroup, resource string) {
	// Reading and arranging what one has queued is arranging one's own
	// conversation; writing a message that will be sent is writing to it,
	// gated like sending one.
	read := h.pm.RequirePermission(resource, permission.OpRead)
	write := h.pm.RequirePermission(resource, permission.OpCreate)

	api.GET("/threads/:threadID/queue/", read, h.listQueue)
	api.POST("/threads/:threadID/queue/", write, h.enqueue)
	api.PUT("/threads/:threadID/queue/order/", read, h.reorderQueue)
	api.PATCH("/threads/:threadID/queue/:itemID/", write, h.editQueued)
	api.DELETE("/threads/:threadID/queue/:itemID/", read, h.removeQueued)
	api.POST("/threads/:threadID/queue/:itemID/send/", write, h.sendQueued)
}

type queueListResponse struct {
	Items []*conversation.QueuedMessage `json:"items"`
}

type enqueueRequest struct {
	Content               string              `json:"content"`
	Context               *pageContextRequest `json:"context"`
	Surface               agent.Surface       `json:"surface"`
	ProviderID            *pulid.ID           `json:"providerId"`
	AttachmentDocumentIDs []pulid.ID          `json:"attachmentDocumentIds"`
	Mentions              []agent.EntityRef   `json:"mentions"`
	// Steer asks for the message to be read into the reply under way at its
	// next step, rather than sent once the reply ends.
	Steer bool `json:"steer"`
}

type editQueuedRequest struct {
	Content string `json:"content"`
	Version int64  `json:"version"`
}

type reorderQueueRequest struct {
	IDs []pulid.ID `json:"ids"`
}

// queueOutcomeResponse is what became of a message handed to the queue: still
// waiting, read into the reply under way, or sent as a turn of its own.
type queueOutcomeResponse struct {
	Item     *conversation.QueuedMessage `json:"item,omitempty"`
	Steering bool                        `json:"steering"`
	Held     string                      `json:"held,omitempty"`
	Turn     *startTurnResponse          `json:"turn,omitempty"`
}

func outcomeResponse(outcome *assistantqueueservice.Outcome) queueOutcomeResponse {
	response := queueOutcomeResponse{
		Item:     outcome.Item,
		Steering: outcome.Steering,
		Held:     outcome.Held,
	}
	if outcome.Turn != nil {
		started := turnStarted(outcome.Turn)
		response.Turn = &started
	}

	return response
}

func queueScope(c *gin.Context) (*assistantqueueservice.Scope, error) {
	threadID, err := pulid.Parse(c.Param("threadID"))
	if err != nil {
		return nil, err
	}

	return &assistantqueueservice.Scope{
		ThreadID: threadID,
		Actor:    requestActorFromAuthContext(authctx.GetAuthContext(c)),
	}, nil
}

func queueItem(c *gin.Context) (*assistantqueueservice.Scope, pulid.ID, error) {
	scope, err := queueScope(c)
	if err != nil {
		return scope, pulid.Nil, err
	}
	itemID, err := pulid.Parse(c.Param("itemID"))

	return scope, itemID, err
}

func (h *Handler) listQueue(c *gin.Context) {
	scope, err := queueScope(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.queue.List(c.Request.Context(), scope)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, queueListResponse{Items: items})
}

func (h *Handler) enqueue(c *gin.Context) {
	scope, err := queueScope(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body enqueueRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	request := conversation.QueuedRequest{
		Page:                  body.Context.page(),
		Surface:               body.Surface,
		Mentions:              body.Mentions,
		AttachmentDocumentIDs: body.AttachmentDocumentIDs,
	}
	if body.ProviderID != nil {
		request.ProviderID = *body.ProviderID
		request.ProviderChosen = true
	}

	outcome, err := h.queue.Enqueue(c.Request.Context(), &assistantqueueservice.EnqueueRequest{
		Scope:   scope,
		Content: body.Content,
		Request: request,
		Steer:   body.Steer,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, outcomeResponse(outcome))
}

func (h *Handler) editQueued(c *gin.Context) {
	scope, itemID, err := queueItem(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body editQueuedRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	item, err := h.queue.Edit(c.Request.Context(), &assistantqueueservice.EditRequest{
		Scope:   scope,
		ID:      itemID,
		Content: body.Content,
		Version: body.Version,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

func (h *Handler) removeQueued(c *gin.Context) {
	scope, itemID, err := queueItem(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.queue.Remove(c.Request.Context(), &assistantqueueservice.ItemRequest{
		Scope: scope,
		ID:    itemID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) reorderQueue(c *gin.Context) {
	scope, err := queueScope(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body reorderQueueRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.queue.Reorder(c.Request.Context(), &assistantqueueservice.ReorderRequest{
		Scope: scope,
		IDs:   body.IDs,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, queueListResponse{Items: items})
}

func (h *Handler) sendQueued(c *gin.Context) {
	scope, itemID, err := queueItem(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	outcome, err := h.queue.SendNow(c.Request.Context(), &assistantqueueservice.ItemRequest{
		Scope: scope,
		ID:    itemID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, outcomeResponse(outcome))
}
