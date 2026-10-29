package capturehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturedevices"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
)

// @Summary Start pairing a Trenova Capture device
// @Description Issues an RFC 8628 device authorization grant. The device shows the user code
// @Description and polls the token endpoint with the device code until a signed-in person
// @Description approves it.
// @ID startCapturePairing
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body capturedevices.StartPairingRequest true "The machine asking to be paired"
// @Success 200 {object} capturedevices.PairingGrant
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 429 {object} helpers.ProblemDetail
// @Router /capture/pair/ [post]
func (h *Handler) startPairing(c *gin.Context) {
	req := new(capturedevices.StartPairingRequest)
	if err := bindJSON(c, req); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	req.ClientIP = c.ClientIP()

	grant, err := h.devices.StartPairing(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)

		return
	}

	helpers.JSON(c, http.StatusOK, grant)
}

type exchangePairingRequest struct {
	DeviceCode string `json:"deviceCode"`
}

// @Summary Exchange an approved pairing for device credentials
// @Description Polled by the device. Answers authorization_pending, slow_down, access_denied,
// @Description expired_token or invalid_grant as an RFC 8628 error body until the grant is
// @Description approved, then returns the device's first credential exactly once.
// @ID exchangeCapturePairing
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body exchangePairingRequest true "The device code"
// @Success 200 {object} capturedevices.TokenPair
// @Failure 400 {object} map[string]string
// @Router /capture/pair/token/ [post]
func (h *Handler) exchangePairing(c *gin.Context) {
	req := new(exchangePairingRequest)
	if err := bindJSON(c, req); err != nil {
		h.eh.HandleError(c, err)

		return
	}

	pair, err := h.devices.ExchangePairing(c.Request.Context(), req.DeviceCode)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, pair)
}

// @Summary Rotate a device's credential
// @Description Exchanges a refresh token for a new access and refresh token. A refresh token
// @Description is single use; presenting one that was already replaced revokes the device.
// @ID refreshCaptureToken
// @Tags Capture
// @Accept json
// @Produce json
// @Param request body capturedevices.RefreshRequest true "The refresh token"
// @Success 200 {object} capturedevices.TokenPair
// @Failure 400 {object} map[string]string
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 426 {object} map[string]string
// @Router /capture/token/refresh/ [post]
func (h *Handler) refreshToken(c *gin.Context) {
	req := new(capturedevices.RefreshRequest)
	if err := bindJSON(c, req); err != nil {
		h.eh.HandleError(c, err)

		return
	}
	req.ClientIP = c.ClientIP()

	pair, err := h.devices.Refresh(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)

		return
	}

	c.Header("Cache-Control", "no-store")
	helpers.JSON(c, http.StatusOK, pair)
}

// releaseCacheSeconds is how long a browser or proxy may keep the manifest.
const releaseCacheSeconds = 300

// @Summary The current Trenova Capture release
// @Description Returns the release manifest exactly as it was signed: its bytes, base64, and
// @Description their ed25519 signature. The companion verifies the signature against a key built
// @Description into it before acting on anything the manifest says. 404 when no release is
// @Description published or this server does not serve one.
// @ID getLatestCaptureRelease
// @Tags Capture
// @Produce json
// @Success 200 {object} capturereleaseservice.SignedRelease
// @Failure 404 {object} helpers.ProblemDetail
// @Router /capture/releases/latest/ [get]
func (h *Handler) latestRelease(c *gin.Context) {
	latest, err := h.releases.Latest(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)

		return
	}
	if latest == nil {
		h.eh.HandleError(c, errortypes.NewNotFoundError("No Trenova Capture release is published"))

		return
	}

	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(releaseCacheSeconds))
	helpers.JSON(c, http.StatusOK, latest.Signed)
}
