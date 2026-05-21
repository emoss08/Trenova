package controlplaneprovisioninghandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

const (
	headerInstanceID = "X-Trenova-Instance-ID"
	headerTimestamp  = "X-Trenova-Timestamp"
	headerBodySHA256 = "X-Trenova-Body-SHA256"
	headerSignature  = "X-Trenova-Signature"

	maxSignatureAge = 5 * time.Minute
)

type Params struct {
	fx.In

	Config       *config.Config
	Service      services.TenantProvisioningService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	cfg     *config.Config
	service services.TenantProvisioningService
	eh      *helpers.ErrorHandler
	now     func() time.Time
}

func New(p Params) *Handler {
	return &Handler{
		cfg:     p.Config,
		service: p.Service,
		eh:      p.ErrorHandler,
		now:     time.Now,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/control-plane/tenants")
	api.POST("/provision", h.provisionTenant)
}

func (h *Handler) provisionTenant(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"payload",
			errortypes.ErrInvalid,
			"Provisioning payload could not be read",
		))
		return
	}

	if err = h.verifySignature(c, body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(services.TenantProvisioningRequest)
	if err = sonic.Unmarshal(body, req); err != nil {
		h.eh.HandleError(c, errortypes.NewValidationError(
			"payload",
			errortypes.ErrInvalid,
			"Provisioning payload is invalid JSON",
		))
		return
	}

	result, err := h.service.ProvisionTenant(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, result)
}

func (h *Handler) verifySignature(c *gin.Context, body []byte) error {
	secret := strings.TrimSpace(h.cfg.Platform.ControlPlane.APIKey)
	if secret == "" {
		return errortypes.NewAuthorizationError("Control-plane shared secret is not configured")
	}

	timestamp := strings.TrimSpace(c.GetHeader(headerTimestamp))
	if timestamp == "" {
		return errortypes.NewAuthorizationError("Control-plane timestamp is required")
	}
	if err := h.verifyTimestamp(timestamp); err != nil {
		return err
	}

	bodyHash := bodySHA256(body)
	if !hmac.Equal(
		[]byte(bodyHash),
		[]byte(strings.TrimSpace(c.GetHeader(headerBodySHA256))),
	) {
		return errortypes.NewAuthorizationError("Control-plane body hash is invalid")
	}

	expected := computeSignature(
		secret,
		c.Request.Method,
		c.Request.URL.Path,
		bodyHash,
		timestamp,
	)
	if !hmac.Equal(
		[]byte(expected),
		[]byte(strings.TrimSpace(c.GetHeader(headerSignature))),
	) {
		return errortypes.NewAuthorizationError("Control-plane signature is invalid")
	}

	instanceID := strings.TrimSpace(c.GetHeader(headerInstanceID))
	if h.cfg.Platform.InstanceID != "" && instanceID != h.cfg.Platform.InstanceID {
		return errortypes.NewAuthorizationError("Control-plane request targets a different instance")
	}

	return nil
}

func (h *Handler) verifyTimestamp(timestamp string) error {
	sentAt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errortypes.NewAuthorizationError("Control-plane timestamp is invalid")
	}

	age := h.now().Sub(time.Unix(sentAt, 0))
	if age < -maxSignatureAge || age > maxSignatureAge {
		return errortypes.NewAuthorizationError("Control-plane signature has expired")
	}

	return nil
}

func bodySHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func computeSignature(secret, method, path, bodyHash, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, method)
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, path)
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, bodyHash)
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, timestamp)
	return hex.EncodeToString(mac.Sum(nil))
}
