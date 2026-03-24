package workerhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerservice"
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
	ErrorHandler         *helpers.ErrorHandler
}

type Handler struct {
	service *workerservice.Service
	pm      *middleware.PermissionMiddleware
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		pm:      p.PermissionMiddleware,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workers")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceWorker.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:workerID/",
		h.pm.RequirePermission(permission.ResourceWorker.String(), permission.OpRead),
		h.get,
	)
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

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:workerID", h.getOption)
}

// @Summary List workers
// @ID listWorkers
// @Tags Workers
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]worker.Worker]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*worker.Worker], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListWorkersRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a worker option
// @ID getWorkerOption
// @Tags Workers
// @Produce json
// @Param workerID path string true "Worker ID"
// @Success 200 {object} worker.Worker
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/select-options/{workerID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetWorkerByIDRequest{
		ID: workerID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List worker options
// @ID listWorkerOptions
// @Tags Workers
// @Produce json
// @Param query query string false "Search query"
// @Param includeProfile query bool false "Include worker profile"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]worker.Worker]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*worker.Worker], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.WorkerSelectOptionsRequest{
					SelectQueryRequest: req,
					IncludeProfile:     helpers.QueryBool(c, "includeProfile", false),
				},
			)
		},
	)
}

// @Summary Get a worker
// @ID getWorker
// @Tags Workers
// @Produce json
// @Param workerID path string true "Worker ID"
// @Success 200 {object} worker.Worker
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /workers/{workerID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	workerID, err := pulid.MustParse(c.Param("workerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetWorkerByIDRequest{
			ID: workerID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			IncludeProfile: true,
			IncludeState:   true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
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
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	created, err := h.service.Create(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
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
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), entity, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
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
		repositories.GetWorkerByIDRequest{
			ID: workerID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			IncludeProfile: true,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = c.ShouldBindJSON(existing); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	actor := actorutil.FromAuthContext(authCtx)
	updated, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
