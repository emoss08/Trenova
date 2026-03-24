package databasesessionhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/system"
	"github.com/emoss08/trenova/internal/core/services/databasesessionservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

var (
	_ *system.DatabaseSessionChain
	_ *system.TerminateDatabaseSessionResult
)

type Params struct {
	fx.In

	Service              *databasesessionservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *databasesessionservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
		pm:      p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/admin/database-sessions")
	api.Use(h.pm.RequirePlatformAdmin())
	api.GET("/", h.listBlocked)
	api.POST("/:pid/terminate/", h.terminate)
}

// @Summary List blocked database sessions
// @Description Returns currently blocked PostgreSQL session chains for platform administrators.
// @ID listBlockedDatabaseSessions
// @Tags Database Sessions
// @Produce json
// @Success 200 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /admin/database-sessions/ [get]
func (h *Handler) listBlocked(c *gin.Context) {
	rows, err := h.service.ListBlocked(c.Request.Context())
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": rows})
}

// @Summary Terminate a database session
// @Description Terminates the specified PostgreSQL session for platform administrators.
// @ID terminateDatabaseSession
// @Tags Database Sessions
// @Produce json
// @Param pid path int true "Database session PID"
// @Success 200 {object} system.TerminateDatabaseSessionResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /admin/database-sessions/{pid}/terminate/ [post]
func (h *Handler) terminate(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("pid"), 10, 64)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	authCtx := authctx.GetAuthContext(c)
	result, err := h.service.TerminateSession(
		c.Request.Context(),
		pid,
		authCtx.UserID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
