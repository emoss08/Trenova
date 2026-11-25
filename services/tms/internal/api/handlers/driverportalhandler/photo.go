package driverportalhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/kernel/app/photoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

const maxPhotoBodyBytes = photoservice.MaxUploadBytes + 1<<20

// @Summary Send a new photo for the driver's record
// @Description Stages the photo and asks the office to approve it; a newer photo replaces one still waiting.
// @ID submitPortalPhoto
// @Tags DriverPortal
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Photo file"
// @Param note formData string false "A note for the office"
// @Success 201 {object} driverportalservice.PortalPhotoChange
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Router /portal/profile/photo [post]
func (h *Handler) submitPhoto(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	file, err := c.FormFile("file")
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"file",
			errortypes.ErrRequired,
			"A photo file is required",
		))
		return
	}

	submitted, err := h.service.SubmitMyPhoto(
		c.Request.Context(),
		&driverportalservice.SubmitMyPhotoRequest{
			TenantInfo: tenantInfoFrom(authCtx),
			File:       file,
			Note:       c.PostForm("note"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusCreated, submitted)
}
