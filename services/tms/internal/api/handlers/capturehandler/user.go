package capturehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
)

// @Summary Preview a pairing code
// @Description Shows which machine is asking to be paired, before the person approves it.
// @ID previewCapturePairing
// @Tags Capture
// @Produce json
// @Param userCode path string true "The code the device shows"
// @Success 200 {object} captureservice.PairingPreview
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/pairings/{userCode}/ [get]
func (h *Handler) previewPairing(c *gin.Context) {
	preview, err := h.service.PreviewPairing(
		c.Request.Context(),
		userTenant(c),
		c.Param("userCode"),
	)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, preview)
}

// @Summary Approve or deny a pairing
// @Description Approval binds the device to the person approving it, in the organization
// @Description they are signed in to.
// @ID decideCapturePairing
// @Tags Capture
// @Accept json
// @Param request body captureservice.DecidePairingRequest true "The decision"
// @Success 204
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/pairings/decide/ [post]
func (h *Handler) decidePairing(c *gin.Context) {
	req := new(captureservice.DecidePairingRequest)
	if err := bindJSON(c, req); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	req.TenantInfo = userTenant(c)

	if err := h.service.DecidePairing(c.Request.Context(), req); err != nil {
		h.fail(c, err)

		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List paired devices
// @Description With mine=true, the caller's own devices; otherwise every device in the
// @Description organization, which takes capture device read.
// @ID listCaptureDevices
// @Tags Capture
// @Produce json
// @Param mine query bool false "Only the caller's own devices"
// @Param status query string false "Active or Revoked"
// @Success 200 {object} pagination.Response[[]capture.CaptureDevice]
// @Failure 403 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/devices/ [get]
func (h *Handler) listDevices(c *gin.Context) {
	status := capture.DeviceStatus(c.Query("status"))
	if status != "" && !status.IsValid() {
		h.eh.HandleError(c, errortypes.NewValidationError("status", errortypes.ErrInvalid,
			"Status must be Active or Revoked"))

		return
	}

	opts := pagination.NewQueryOptions(c, authctx.GetAuthContext(c))
	pagination.List(c, opts, h.eh, func() (*pagination.ListResult[*capture.CaptureDevice], error) {
		return h.service.ListDevices(c.Request.Context(), &captureservice.ListDevicesRequest{
			TenantInfo: userTenant(c),
			Filter:     opts,
			Mine:       c.Query("mine") == "true",
			Status:     status,
		})
	})
}

type revokeDeviceRequest struct {
	Reason string `json:"reason"`
}

// @Summary Revoke a paired device
// @Description Anyone may revoke their own device; revoking somebody else's takes capture
// @Description device update. The device's stream is closed and its credential stops working.
// @ID revokeCaptureDevice
// @Tags Capture
// @Accept json
// @Produce json
// @Param deviceID path string true "Device ID"
// @Param request body revokeDeviceRequest false "Why"
// @Success 200 {object} capture.CaptureDevice
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/devices/{deviceID}/revoke/ [post]
func (h *Handler) revokeDevice(c *gin.Context) {
	deviceID, err := pathID(c, "deviceID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	req := new(revokeDeviceRequest)
	if c.Request.ContentLength > 0 {
		if err = bindJSON(c, req); err != nil {
			h.eh.HandleError(c, err)

			return
		}
	}

	device, err := h.service.RevokeDevice(c.Request.Context(), &captureservice.RevokeDeviceRequest{
		TenantInfo: userTenant(c),
		DeviceID:   deviceID,
		Reason:     req.Reason,
	})
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, device)
}

// @Summary Ask a device to capture into a record
// @Description Sends a scan to one of the caller's own devices, or arms the next print from it,
// @Description filed onto the record given.
// @ID createCaptureRequest
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body captureservice.CreateRequestInput true "What to capture, and where"
// @Success 201 {object} capture.CaptureRequest
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/requests/ [post]
func (h *Handler) createRequest(c *gin.Context) {
	in := new(captureservice.CreateRequestInput)
	if err := bindJSON(c, in); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	in.TenantInfo = userTenant(c)

	req, err := h.service.CreateRequest(c.Request.Context(), in)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusCreated, req)
}

// @Summary Cancel a capture request
// @ID cancelCaptureRequest
// @Tags Capture
// @Produce json
// @Param requestID path string true "Capture request ID"
// @Success 200 {object} capture.CaptureRequest
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/requests/{requestID}/cancel/ [post]
func (h *Handler) cancelRequest(c *gin.Context) {
	requestID, err := pathID(c, "requestID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	req, err := h.service.CancelRequest(c.Request.Context(), userTenant(c), requestID)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, req)
}

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

	content, err := h.service.PageContent(c.Request.Context(), userTenant(c), pageID, kind)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.Header("Cache-Control", "private, max-age=300")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, content.ContentType, content.Data)
}
