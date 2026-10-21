package capturehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
)

const (
	// DiscoveryProduct names what answers at the discovery route, so a
	// companion pointed at some other web server knows it is not a Trenova.
	DiscoveryProduct = "trenova"
	// DiscoveryWellKnownPath is where the web app publishes the address of
	// its API, for a companion given the address people open in a browser.
	DiscoveryWellKnownPath = "/.well-known/trenova-capture.json"

	discoveryCacheSeconds = 300
	cloudServerName       = "Trenova Cloud"
	selfHostedServerName  = "Trenova"
)

// Discovery is what a Trenova Capture companion learns about a server before
// it is paired: which edition answers, what the server calls itself, where
// the web app is, and whether computers can be paired here at all.
type Discovery struct {
	Product       string `json:"product"`
	Edition       string `json:"edition"`
	SharedTenancy bool   `json:"sharedTenancy"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	WebBaseURL    string `json:"webBaseUrl"`
	Pairing       bool   `json:"pairing"`
}

// @Summary Describe this server to a Trenova Capture companion
// @Description Answers, without authentication, which Trenova this is: its edition, the
// @Description name it goes by, the web app's address and whether computers can be paired
// @Description here. The companion calls it with the address a person typed, or the API
// @Description address the web app publishes at /.well-known/trenova-capture.json, to tell a
// @Description Trenova server from any other web server before it starts pairing.
// @ID getCaptureDiscovery
// @Tags Capture
// @Produce json
// @Success 200 {object} Discovery
// @Router /capture/discovery/ [get]
func (h *Handler) discovery(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age="+strconv.Itoa(discoveryCacheSeconds))
	helpers.JSON(c, http.StatusOK, h.Discovery())
}

// Discovery is the server as the discovery route describes it.
func (h *Handler) Discovery() Discovery {
	name := selfHostedServerName
	if h.edition.SharedTenancy() {
		name = cloudServerName
	}

	return Discovery{
		Product:       DiscoveryProduct,
		Edition:       h.edition.Name(),
		SharedTenancy: h.edition.SharedTenancy(),
		Name:          name,
		Version:       h.version,
		WebBaseURL:    h.webBase,
		Pairing:       h.webBase != "",
	}
}
