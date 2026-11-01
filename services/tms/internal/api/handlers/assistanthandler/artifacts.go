package assistanthandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func artifactParam(c *gin.Context) (pulid.ID, error) {
	return pulid.Parse(c.Param("artifactID"))
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

	name, err := h.artifacts.ArtifactCSVName(c.Request.Context(), &req, artifactID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(name))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if err = h.artifacts.ExportArtifactCSV(
		c.Request.Context(),
		&req,
		artifactID,
		c.Writer,
	); err != nil {
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

	file, err := h.artifacts.ExportDocument(
		c.Request.Context(),
		&req,
		artifactID,
		c.Query("format"),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+strconv.Quote(file.FileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, file.ContentType, file.Body)
}
