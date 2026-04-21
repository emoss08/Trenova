package versionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/versionservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var (
	_ *serviceports.VersionInfo
	_ *serviceports.UpdateStatus
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

// @Summary Get version information
// @Description Returns the current service version and build metadata.
// @ID getVersion
// @Tags System
// @Produce json
// @Success 200 {object} services.VersionInfo
// @Failure 500 {object} helpers.ProblemDetail
// @Router /system/version [get]
func (h *Handler) getVersion(c *gin.Context) {
	info, err := h.service.GetVersionInfo(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, info)
}

// @Summary Get update status
// @Description Returns the cached update check status for the running service.
// @ID getUpdateStatus
// @Tags System
// @Produce json
// @Success 200 {object} services.UpdateStatus
// @Failure 500 {object} helpers.ProblemDetail
// @Router /system/update-status [get]
func (h *Handler) getUpdateStatus(c *gin.Context) {
	status, err := h.service.GetUpdateStatus(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

// @Summary Check for updates
// @Description Performs an update check and returns the latest status.
// @ID checkForUpdates
// @Tags System
// @Produce json
// @Success 200 {object} services.UpdateStatus
// @Failure 500 {object} helpers.ProblemDetail
// @Router /system/check-updates [post]
func (h *Handler) checkUpdates(c *gin.Context) {
	status, err := h.service.CheckForUpdates(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}
