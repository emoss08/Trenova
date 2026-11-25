package workerhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/kernel/app/photoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxPhotoBodyBytes = photoservice.MaxUploadBytes + 1<<20

type PhotoResponse struct {
	URL          *string `json:"url"`
	ThumbnailURL *string `json:"thumbnailUrl"`
	Version      int64   `json:"version"`
}

func photoResponse(result *workerphotoservice.Result) PhotoResponse {
	response := PhotoResponse{Version: result.Version}
	if result.Photo != nil {
		response.URL = &result.Photo.URL
		response.ThumbnailURL = &result.Photo.ThumbnailURL
	}
	return response
}

func photoVersion(raw string) (int64, error) {
	version, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || version < 0 {
		return 0, errortypes.NewValidationError(
			"version",
			errortypes.ErrRequired,
			"The worker's version is required",
		)
	}
	return version, nil
}

// @Summary Upload a worker's photo
// @ID uploadWorkerPhoto
// @Tags Workers
// @Accept mpfd
// @Produce json
// @Param workerID path string true "Worker ID"
// @Param file formData file true "Photo file"
// @Param version formData int true "The worker's version"
// @Success 200 {object} workerhandler.PhotoResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/{workerID}/photo/ [post]
func (h *Handler) uploadPhoto(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := photoVersion(c.PostForm("version"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"file",
			errortypes.ErrRequired,
			"A photo file is required",
		))
		return
	}

	result, err := h.photos.Upload(c.Request.Context(), &workerphotoservice.UploadRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		WorkerID:   workerID,
		Version:    version,
		File:       file,
		Actor:      actorutil.FromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, photoResponse(result))
}

// @Summary Remove a worker's photo
// @ID removeWorkerPhoto
// @Tags Workers
// @Produce json
// @Param workerID path string true "Worker ID"
// @Param version query int true "The worker's version"
// @Success 200 {object} workerhandler.PhotoResponse
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/{workerID}/photo/ [delete]
func (h *Handler) removePhoto(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	version, err := photoVersion(c.Query("version"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.photos.Remove(c.Request.Context(), &workerphotoservice.RemoveRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		WorkerID:   workerID,
		Version:    version,
		Actor:      actorutil.FromAuthContext(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, photoResponse(result))
}
