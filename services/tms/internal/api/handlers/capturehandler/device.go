package capturehandler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

const (
	principalKey     = "capturePrincipal"
	dpiHeader        = "X-Capture-Dpi"
	patchCodeHeader  = "X-Capture-Patch-Code"
	barcodeHeader    = "X-Capture-Barcode"
	maxPrintJobBytes = 200 << 20
	bearerPrefix     = "bearer "
)

// RequireDevice authenticates a device access token and nothing else. It
// sets the auth context to the device acting for its person, so the rate
// limiter, the control plane and every permission check see that person.
func (h *Handler) RequireDevice() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if len(header) <= len(bearerPrefix) ||
			!strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
			h.eh.HandleError(
				c,
				errortypes.NewAuthenticationError("A device credential is required"),
			)
			c.Abort()

			return
		}

		principal, err := h.service.Authenticate(
			c.Request.Context(),
			strings.TrimSpace(header[len(bearerPrefix):]),
			c.ClientIP(),
		)
		if err != nil {
			h.eh.HandleError(c, err)
			c.Abort()

			return
		}

		device := principal.Device
		authctx.SetCaptureDeviceContext(
			c,
			device.ID,
			device.UserID,
			device.BusinessUnitID,
			device.OrganizationID,
		)
		c.Set(principalKey, principal)
		c.Next()
	}
}

func devicePrincipal(c *gin.Context) *captureservice.DevicePrincipal {
	value, _ := c.Get(principalKey)
	principal, _ := value.(*captureservice.DevicePrincipal)

	return principal
}

// @Summary Describe the calling device
// @Description Returns the device record, including the person it acts for, so the tray can
// @Description show who it is signed in as.
// @ID getCaptureDevice
// @Tags Capture
// @Produce json
// @Success 200 {object} capture.CaptureDevice
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/ [get]
func (h *Handler) me(c *gin.Context) {
	c.JSON(http.StatusOK, devicePrincipal(c).Device)
}

type reportSourcesRequest struct {
	Sources []capture.SourceInfo `json:"sources"`
}

// @Summary Report the scanners a device can reach
// @ID reportCaptureSources
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body reportSourcesRequest true "Every scanner the device found"
// @Success 200 {object} capture.CaptureDevice
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/sources/ [put]
func (h *Handler) reportSources(c *gin.Context) {
	req := new(reportSourcesRequest)
	if err := bindJSON(c, req); err != nil {
		h.eh.HandleError(c, err)

		return
	}

	device, err := h.service.ReportSources(c.Request.Context(), devicePrincipal(c), req.Sources)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, device)
}

// @Summary List what the device still has to do
// @Description Returns the device's open capture requests, oldest first. Requests whose time
// @Description ran out are expired on the way and not returned.
// @ID listCaptureDeviceRequests
// @Tags Capture
// @Produce json
// @Success 200 {array} capture.CaptureRequest
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/requests/ [get]
func (h *Handler) openRequests(c *gin.Context) {
	requests, err := h.service.OpenRequests(c.Request.Context(), devicePrincipal(c))
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, requests)
}

// @Summary Report how a capture request is going
// @ID reportCaptureRequestStatus
// @Tags Capture
// @Accept json
// @Produce json
// @Param requestID path string true "Capture request ID"
// @Param request body captureservice.RequestStatusReport true "The new status"
// @Success 200 {object} capture.CaptureRequest
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/requests/{requestID}/status/ [post]
func (h *Handler) reportRequestStatus(c *gin.Context) {
	requestID, err := pathID(c, "requestID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	report := new(captureservice.RequestStatusReport)
	if err = bindJSON(c, report); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	report.RequestID = requestID

	updated, err := h.service.ReportRequestStatus(c.Request.Context(), devicePrincipal(c), report)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Open a capture batch
// @Description Opening twice with the same client key returns the same batch, so a retried
// @Description open after a lost response does not start a second batch.
// @ID openCaptureBatch
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body captureservice.OpenBatchInput true "The batch to open"
// @Success 200 {object} capture.CaptureBatch
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 426 {object} map[string]string
// @Security BearerAuth
// @Router /capture/device/batches/ [post]
func (h *Handler) openBatch(c *gin.Context) {
	in := new(captureservice.OpenBatchInput)
	if err := bindJSON(c, in); err != nil {
		h.eh.HandleError(c, err)

		return
	}

	batch, err := h.service.OpenBatch(c.Request.Context(), devicePrincipal(c), in)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, batch)
}

// @Summary Upload one scanned page
// @Description The body is a one-page PDF. Idempotent on the sequence: the same page sent
// @Description again returns the stored page, and a different page for a sequence already
// @Description taken is refused. Scanner-reported markers travel in headers.
// @ID putCapturePage
// @Tags Capture
// @Accept application/pdf
// @Produce json
// @Param batchID path string true "Capture batch ID"
// @Param sequence path int true "Page sequence, from 1"
// @Param X-Capture-Dpi header int false "Resolution the page was scanned at"
// @Param X-Capture-Patch-Code header string false "Patch sheet the scanner detected"
// @Param X-Capture-Barcode header string false "A barcode the scanner decoded; repeatable"
// @Success 200 {object} capture.CapturePage
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 413 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/batches/{batchID}/pages/{sequence}/ [put]
func (h *Handler) putPage(c *gin.Context) {
	batchID, err := pathID(c, "batchID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	sequence, err := strconv.Atoi(c.Param("sequence"))
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError("sequence", errortypes.ErrInvalid,
			"Page sequence must be a number"))

		return
	}

	dpi := 0
	if raw := strings.TrimSpace(c.GetHeader(dpiHeader)); raw != "" {
		if dpi, err = strconv.Atoi(raw); err != nil || dpi < 0 {
			h.eh.HandleError(c, errortypes.NewValidationError("dpi", errortypes.ErrInvalid,
				"Resolution must be a whole number"))

			return
		}
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, capture.MaxPageBytes+1)
	page, err := h.service.PutPage(
		c.Request.Context(),
		devicePrincipal(c),
		&captureservice.PutPageInput{
			BatchID:        batchID,
			Sequence:       sequence,
			Body:           c.Request.Body,
			DPI:            dpi,
			PatchCode:      c.GetHeader(patchCodeHeader),
			DeviceBarcodes: c.Request.Header.Values(barcodeHeader),
		},
	)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, page)
}

// @Summary Upload a whole print job
// @Description The body is the job as a PDF. It is split into pages and the batch is sealed.
// @ID putCapturePrintJob
// @Tags Capture
// @Accept application/pdf
// @Produce json
// @Param batchID path string true "Capture batch ID"
// @Success 200 {object} capture.CaptureBatch
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 413 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/batches/{batchID}/print-job/ [put]
func (h *Handler) putPrintJob(c *gin.Context) {
	batchID, err := pathID(c, "batchID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPrintJobBytes+1)
	batch, err := h.service.PutPrintJob(
		c.Request.Context(),
		devicePrincipal(c),
		&captureservice.PutPrintJobInput{
			BatchID: batchID,
			Body:    c.Request.Body,
		},
	)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, batch)
}

// @Summary Seal a capture batch
// @Description Closes the batch to new pages once every page is stored and the manifest digest
// @Description matches, then starts reading it. Sealing a sealed batch returns it unchanged.
// @ID sealCaptureBatch
// @Tags Capture
// @Accept json
// @Produce json
// @Param batchID path string true "Capture batch ID"
// @Param request body captureservice.SealBatchInput true "Page count and manifest digest"
// @Success 200 {object} capture.CaptureBatch
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/batches/{batchID}/seal/ [post]
func (h *Handler) sealBatch(c *gin.Context) {
	batchID, err := pathID(c, "batchID")
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}

	in := new(captureservice.SealBatchInput)
	if err = bindJSON(c, in); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	in.BatchID = batchID

	batch, err := h.service.SealBatch(c.Request.Context(), devicePrincipal(c), in)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.JSON(http.StatusOK, batch)
}
