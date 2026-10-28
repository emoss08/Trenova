package assistanthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

// A conversation's chapters are its owner arranging their own conversation,
// read and changed under their user id like naming the conversation.
func (h *Handler) registerChapterRoutes(api *gin.RouterGroup, resource string) {
	read := h.pm.RequirePermission(resource, permission.OpRead)

	api.GET("/threads/:threadID/chapters/", read, h.listChapters)
	api.POST("/threads/:threadID/chapters/", read, h.openChapter)
	api.PATCH("/threads/:threadID/chapters/:chapterID/", read, h.renameChapter)
	api.DELETE("/threads/:threadID/chapters/:chapterID/", read, h.removeChapter)
}

type chapterListResponse struct {
	Items []*conversation.Chapter `json:"items"`
}

type openChapterRequest struct {
	MessageID pulid.ID `json:"messageId" binding:"required"`
}

type renameChapterRequest struct {
	Title   string `json:"title"`
	Version *int64 `json:"version"`
}

func (h *Handler) listChapters(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	items, err := h.chapters.List(c.Request.Context(), agentports.ThreadChaptersRequest{
		ThreadID:   req.ID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if items == nil {
		items = []*conversation.Chapter{}
	}

	helpers.JSON(c, http.StatusOK, chapterListResponse{Items: items})
}

func (h *Handler) openChapter(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body openChapterRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	chapter, err := h.chapters.Open(c.Request.Context(), agentports.OpenChapterRequest{
		ThreadID:   req.ID,
		UserID:     req.UserID,
		MessageID:  body.MessageID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, chapter)
}

func (h *Handler) renameChapter(c *gin.Context) {
	req, chapterID, err := chapterRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body renameChapterRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	chapter, err := h.chapters.Rename(c.Request.Context(), agentports.RenameChapterRequest{
		ID:         chapterID,
		ThreadID:   req.ID,
		UserID:     req.UserID,
		Title:      body.Title,
		Version:    body.Version,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, chapter)
}

func (h *Handler) removeChapter(c *gin.Context) {
	req, chapterID, err := chapterRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.chapters.Remove(c.Request.Context(), agentports.RemoveChapterRequest{
		ID:         chapterID,
		ThreadID:   req.ID,
		UserID:     req.UserID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func chapterRequest(c *gin.Context) (agentports.GetThreadRequest, pulid.ID, error) {
	req, err := threadRequest(c)
	if err != nil {
		return req, pulid.Nil, err
	}
	chapterID, err := pulid.Parse(c.Param("chapterID"))

	return req, chapterID, err
}
