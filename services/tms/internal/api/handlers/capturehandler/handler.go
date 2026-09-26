// Package capturehandler is the HTTP surface of Trenova Capture.
//
// It has three audiences, mounted separately because each is authenticated
// differently:
//
//   - The pairing and token endpoints are public. A machine asking to be
//     paired cannot yet prove who it is; the grant it gets carries no tenant
//     until a signed-in person approves it, and a refresh token is its own
//     proof.
//   - The device endpoints take a device access token and nothing else: a
//     session cookie is never accepted there, and a device token is never
//     accepted anywhere else.
//   - The approval and management endpoints are ordinary signed-in routes.
package capturehandler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxJSONBodyBytes bounds every JSON body here. Nothing the companion or
	// the web app sends as JSON is more than a few kilobytes.
	maxJSONBodyBytes = 256 << 10
	// retryAfterSeconds is how long a device waits after a refused stream.
	retryAfterSeconds = 5
)

type Params struct {
	fx.In

	Service      *captureservice.Service
	Gateway      services.RealtimeGateway `optional:"true"`
	ErrorHandler *helpers.ErrorHandler
	Config       *config.Config
	Logger       *zap.Logger
}

type Handler struct {
	service     *captureservice.Service
	gateway     services.RealtimeGateway
	eh          *helpers.ErrorHandler
	maxLifetime int64
	l           *zap.Logger
}

func New(p Params) *Handler {
	return &Handler{
		service:     p.Service,
		gateway:     p.Gateway,
		eh:          p.ErrorHandler,
		maxLifetime: int64(p.Config.GetRealtimeConfig().GetMaxStreamLifetime().Seconds()),
		l:           p.Logger.Named("handler.capture"),
	}
}

// RegisterPublicRoutes mounts pairing and token refresh.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/capture/")
	api.POST("pair/", h.startPairing)
	api.POST("pair/token/", h.exchangePairing)
	api.POST("token/refresh/", h.refreshToken)
}

// RegisterDeviceRoutes mounts what a paired device calls. The group must
// already carry RequireDevice.
func (h *Handler) RegisterDeviceRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/capture/device/")
	api.GET("", h.me)
	api.PUT("sources/", h.reportSources)
	api.GET("stream/", h.stream)
	api.GET("requests/", h.openRequests)
	api.POST("requests/:requestID/status/", h.reportRequestStatus)
	api.POST("batches/", h.openBatch)
	api.PUT("batches/:batchID/pages/:sequence/", h.putPage)
	api.PUT("batches/:batchID/print-job/", h.putPrintJob)
	api.POST("batches/:batchID/seal/", h.sealBatch)
}

// RegisterRoutes mounts what a signed-in person calls.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/capture/")
	api.GET("pairings/:userCode/", h.previewPairing)
	api.POST("pairings/decide/", h.decidePairing)
	api.GET("devices/", h.listDevices)
	api.POST("devices/:deviceID/revoke/", h.revokeDevice)
	api.POST("requests/", h.createRequest)
	api.POST("requests/:requestID/cancel/", h.cancelRequest)
	api.GET("pages/:pageID/content/", h.pageContent)
}

// fail answers a capture error. The two errors the companion acts on get
// their own status: an RFC 8628 error body it understands without parsing
// problem details, and 426 when it has to update before it can go on.
func (h *Handler) fail(c *gin.Context, err error) {
	var pairingErr *captureservice.PairingError
	if errors.As(err, &pairingErr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": string(pairingErr.Code)})

		return
	}

	var outdated *captureservice.OutdatedAgentError
	if errors.As(err, &outdated) {
		c.JSON(http.StatusUpgradeRequired, gin.H{
			"error":          "agent_outdated",
			"message":        outdated.Error(),
			"minimumVersion": outdated.Minimum,
		})

		return
	}

	h.eh.HandleError(c, err)
}

func bindJSON(c *gin.Context, out any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)

	return c.ShouldBindJSON(out)
}

func pathID(c *gin.Context, name string) (pulid.ID, error) {
	return pulid.MustParse(c.Param(name))
}

func userTenant(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)

	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

func retryAfter(c *gin.Context) {
	c.Header("Retry-After", strconv.Itoa(retryAfterSeconds))
}
