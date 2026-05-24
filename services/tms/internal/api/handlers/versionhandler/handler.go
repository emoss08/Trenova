package versionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/services/versionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *versionservice.Service
	Config               *config.Config
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *versionservice.Service
	cfg     *config.Config
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		cfg:     p.Config,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/system")
	api.GET("/version", h.getVersion)
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/system")
	api.GET("/version", h.getVersion)
	api.GET("/update-status", h.getUpdateStatus)
	api.POST("/check-updates", h.checkUpdates)
}

func (h *Handler) getVersion(c *gin.Context) {
	info, err := h.service.GetVersionInfo(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, info)
}

func (h *Handler) getUpdateStatus(c *gin.Context) {
	status, err := h.service.GetUpdateStatus(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *Handler) checkUpdates(c *gin.Context) {
	status, err := h.service.CheckForUpdates(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}
