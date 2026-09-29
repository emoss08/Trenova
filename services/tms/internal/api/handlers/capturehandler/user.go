package capturehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

// @Summary Read a captured page
// @Description Pages are encrypted at rest, so they are served through the API. kind is pdf
// @Description (the default) or thumbnail.
// @ID getCapturePageContent
// @Tags Capture
// @Produce application/pdf
// @Produce image/jpeg
// @Param pageID path string true "Page ID"
// @Param kind query string false "pdf or thumbnail"
// @Success 200 {file} file
// @Success 304 "The reader already holds this page"
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/pages/{pageID}/content/ [get]
func (h *Handler) pageContent(c *gin.Context) {
	pageID, err := pathID(c, "pageID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	kind := captureservice.PageContentKind(
		c.DefaultQuery("kind", string(captureservice.PageContentPDF)),
	)
	if kind != captureservice.PageContentPDF && kind != captureservice.PageContentThumbnail {
		h.eh.HandleError(c, errortypes.NewValidationError("kind", errortypes.ErrInvalid,
			"kind must be pdf or thumbnail"))

		return
	}

	content, err := h.service.PageContent(c.Request.Context(), &captureservice.PageContentRequest{
		TenantInfo:  userTenant(c),
		PageID:      pageID,
		Kind:        kind,
		IfNoneMatch: c.GetHeader("If-None-Match"),
	})
	if err != nil {
		h.fail(c, err)

		return
	}

	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("ETag", content.ETag)
	c.Header("X-Content-Type-Options", "nosniff")
	if content.NotModified {
		c.Status(http.StatusNotModified)

		return
	}
	c.Data(http.StatusOK, content.ContentType, content.Data)
}
