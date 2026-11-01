package workerhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/fieldguard"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *workerservice.Service
	PermissionMiddleware *middleware.PermissionMiddleware
	PermissionEngine     permissionports.PermissionEngine
	ErrorHandler         *helpers.ErrorHandler
}

type Handler struct {
	service *workerservice.Service
	pm      *middleware.PermissionMiddleware
	engine  permissionports.PermissionEngine
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		pm:      p.PermissionMiddleware,
		engine:  p.PermissionEngine,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) withhold(c *gin.Context, entities ...*worker.Worker) {
	fieldguard.Withhold(fieldguard.Visible(c, h.engine), entities...)
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workers")
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceWorker.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:workerID/",
		h.pm.RequirePermission(permission.ResourceWorker.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:workerID/",
		h.pm.RequirePermission(permission.ResourceWorker.String(), permission.OpUpdate),
		h.patch,
	)
}

// @Summary Create a worker
// @ID createWorker
// @Tags Workers
// @Accept json
// @Produce json
// @Param request body worker.Worker true "Worker payload"
// @Success 201 {object} worker.Worker
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(worker.Worker)
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.withhold(c, created)
	helpers.JSON(c, http.StatusCreated, created)
}

// @Summary Update a worker
// @ID updateWorker
// @Tags Workers
// @Accept json
// @Produce json
// @Param workerID path string true "Worker ID"
// @Param request body worker.Worker true "Worker payload"
// @Success 200 {object} worker.Worker
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/{workerID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(worker.Worker)
	entity.ID = workerID
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.withhold(c, updated)
	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Patch a worker
// @ID patchWorker
// @Tags Workers
// @Accept json
// @Produce json
// @Param workerID path string true "Worker ID"
// @Param request body worker.Worker true "Partial worker payload"
// @Success 200 {object} worker.Worker
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/{workerID} [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		workerports.GetWorkerByIDRequest{
			ID:             workerID,
			TenantInfo:     pagination.FromAuthAsUser(authCtx),
			IncludeProfile: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = authctx.BindJSON(c, authCtx, existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	h.withhold(c, updated)
	helpers.JSON(c, http.StatusOK, updated)
}
