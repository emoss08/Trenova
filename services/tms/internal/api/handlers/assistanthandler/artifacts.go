package assistanthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// registerArtifactRoutes adds the routes a conversation's artifacts are read,
// edited and downloaded through. Each is the person's own conversation, read
// under their user id, so reading and editing what it made needs no more than
// using the assistant: editing a document is arranging their own workspace.
// A download of a table re-reads its source under their own permissions.
func (h *Handler) registerArtifactRoutes(api *gin.RouterGroup, resource string) {
	read := h.pm.RequirePermission(resource, permission.OpRead)

	api.GET("/threads/:threadID/artifacts/", read, h.listThreadArtifacts)
	api.GET("/threads/:threadID/artifacts/by-slug/:slug/", read, h.artifactBySlug)
	api.GET("/threads/:threadID/artifacts/:artifactID/", read, h.artifactLineage)
	api.GET("/threads/:threadID/artifacts/:artifactID/export.csv", read, h.exportArtifactCSV)
	api.GET("/threads/:threadID/artifacts/:artifactID/export/", read, h.exportDocument)
	api.POST("/threads/:threadID/artifacts/:artifactID/versions/", read, h.saveDocumentVersion)
	api.POST("/threads/:threadID/artifacts/:artifactID/restore/", read, h.restoreDocumentVersion)
	api.POST("/threads/:threadID/artifacts/:artifactID/rewrite/", read, h.rewriteDocument)
}

type listArtifactsQuery struct {
	Limit   int    `form:"limit"`
	Cursor  string `form:"cursor"`
	Query   string `form:"q"`
	Kind    string `form:"kind"`
	Pinned  bool   `form:"pinned"`
	Summary bool   `form:"summary"`
}

func (h *Handler) listThreadArtifacts(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var query listArtifactsQuery
	if err = c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	page, err := h.service.ListThreadArtifacts(c.Request.Context(), req, serviceports.ListArtifactsOptions{
		Limit:      query.Limit,
		Cursor:     query.Cursor,
		Query:      query.Query,
		Family:     assistantartifact.Family(query.Kind),
		PinnedOnly: query.Pinned,
		Summary:    query.Summary,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, page)
}

func artifactParam(c *gin.Context) (pulid.ID, error) {
	return pulid.Parse(c.Param("artifactID"))
}

func (h *Handler) artifactLineage(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versions, err := h.service.ArtifactLineage(c.Request.Context(), req, artifactID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": versions})
}

func (h *Handler) artifactBySlug(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	versions, err := h.service.ArtifactBySlug(c.Request.Context(), req, c.Param("slug"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": versions})
}

// exportArtifactCSV streams the table as it is read. The name and the checks
// come first, so a refusal is still an ordinary error response rather than a
// half-written file.
func (h *Handler) exportArtifactCSV(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	name, err := h.service.ArtifactCSVName(c.Request.Context(), req, artifactID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(name))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if err = h.service.ExportArtifactCSV(c.Request.Context(), req, artifactID, c.Writer); err != nil {
		// The status is already sent; the file ends short and the log says why.
		h.logger.Warn("artifact export ended early",
			zap.String("artifactId", artifactID.String()),
			zap.Error(err))
	}
}

func (h *Handler) exportDocument(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	file, err := h.service.ExportDocument(c.Request.Context(), req, artifactID, c.Query("format"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(file.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, file.ContentType, file.Body)
}

func (h *Handler) saveDocumentVersion(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body serviceports.SaveDocumentVersionRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	saved, err := h.service.SaveDocumentVersion(c.Request.Context(), req, artifactID, body)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, saved)
}

func (h *Handler) restoreDocumentVersion(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	saved, err := h.service.RestoreDocumentVersion(c.Request.Context(), req, artifactID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, saved)
}

func (h *Handler) rewriteDocument(c *gin.Context) {
	req, err := threadRequest(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	artifactID, err := artifactParam(c)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var body serviceports.DocumentRewriteRequest
	if err = c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	suggestion, err := h.service.RewriteDocument(c.Request.Context(), req, artifactID, body)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, suggestion)
}
