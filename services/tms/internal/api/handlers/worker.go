package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	workerservice "github.com/emoss08/trenova/internal/core/services/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type WorkerHandlerParams struct {
	fx.In

	Service      *workerservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type WorkerHandler struct {
	service      *workerservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewWorkerHandler(p WorkerHandlerParams) *WorkerHandler {
	return &WorkerHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *WorkerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/workers/")
	api.GET("", h.pm.RequirePermission(permission.ResourceWorker, "read"), h.list)
	api.GET(
		"upcoming-pto/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO, "read"),
		h.listUpcomingPTO,
	)
	api.GET("pto/", h.pm.RequirePermission(permission.ResourceWorkerPTO, "read"), h.listWorkerPTO)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceWorker, "read"), h.get)
	api.GET(
		"pto-chart-data/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO, "read"),
		h.getPTOChartData,
	)
	api.GET(
		"pto-calendar-data/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO, "read"),
		h.getPTOCalendarData,
	)
	api.POST("", h.pm.RequirePermission(permission.ResourceWorker, "create"), h.create)
	api.POST(
		"pto/create/",
		h.pm.RequirePermission(permission.ResourceWorker, "create"),
		h.createPTO,
	)
	api.PUT(
		"pto/:id/reject/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO, "reject"),
		h.rejectPTO,
	)
	api.PUT(
		"pto/:id/approve/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO, "approve"),
		h.approvePTO,
	)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceWorker, "update"), h.update)
}

func (h *WorkerHandler) list(c *gin.Context) {
	pagination.Handle[*worker.Worker](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.Worker], error) {
			return h.service.List(c.Request.Context(), &repositories.ListWorkerRequest{
				Filter: opts,
				WorkerFilterOptions: repositories.WorkerFilterOptions{
					IncludeProfile: helpers.QueryBool(c, "includeProfile"),
					IncludePTO:     helpers.QueryBool(c, "includePTO"),
					Status:         c.Query("status"),
				},
			})
		})
}

func (h *WorkerHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetWorkerByIDRequest{
			WorkerID: id,
			OrgID:    authCtx.OrganizationID,
			BuID:     authCtx.BusinessUnitID,
			UserID:   authCtx.UserID,
			FilterOptions: repositories.WorkerFilterOptions{
				IncludeProfile: helpers.QueryBool(c, "includeProfile"),
				IncludePTO:     helpers.QueryBool(c, "includePTO"),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *WorkerHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(worker.Worker)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *WorkerHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(worker.Worker)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *WorkerHandler) listUpcomingPTO(c *gin.Context) {
	pagination.Handle[*worker.WorkerPTO](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.WorkerPTO], error) {
			return h.service.ListUpcomingPTO(
				c.Request.Context(),
				&repositories.ListUpcomingWorkerPTORequest{
					Filter: opts,
					ListWorkerPTOFilterOptions: repositories.ListWorkerPTOFilterOptions{
						Status:      helpers.QueryString(c, "status"),
						Type:        helpers.QueryString(c, "type"),
						StartDate:   helpers.QueryInt64(c, "startDate"),
						EndDate:     helpers.QueryInt64(c, "endDate"),
						WorkerID:    helpers.QueryString(c, "workerId"),
						FleetCodeID: helpers.QueryString(c, "fleetCodeId"),
					},
				},
			)
		})
}

func (h *WorkerHandler) approvePTO(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.ApprovePTO(c.Request.Context(), &repositories.ApprovePTORequest{
		PtoID:      id,
		BuID:       authCtx.BusinessUnitID,
		OrgID:      authCtx.OrganizationID,
		ApproverID: authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *WorkerHandler) rejectPTO(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req := new(repositories.RejectPTORequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	req.PtoID = id
	req.RejectorID = authCtx.UserID
	context.AddContextToRequest(authCtx, req)

	err = h.service.RejectPTO(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *WorkerHandler) listWorkerPTO(c *gin.Context) {
	pagination.Handle[*worker.WorkerPTO](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.WorkerPTO], error) {
			return h.service.ListWorkerPTO(c.Request.Context(), &repositories.ListWorkerPTORequest{
				Filter: opts,
			})
		})
}

func (h *WorkerHandler) getPTOChartData(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := &repositories.PTOChartDataRequest{
		Filter: &pagination.QueryOptions{
			TenantOpts: pagination.TenantOptions{
				BuID:   authCtx.BusinessUnitID,
				OrgID:  authCtx.OrganizationID,
				UserID: authCtx.UserID,
			},
		},
		StartDate: helpers.QueryInt64(c, "startDate"),
		EndDate:   helpers.QueryInt64(c, "endDate"),
		Type:      helpers.QueryString(c, "type"),
		Timezone:  helpers.QueryString(c, "timezone"),
		WorkerID:  helpers.QueryString(c, "workerId"),
	}

	if err := c.ShouldBindQuery(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	ptoData, err := h.service.GetPTOChartData(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ptoData)
}

func (h *WorkerHandler) getPTOCalendarData(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := &repositories.PTOCalendarDataRequest{
		Filter: &pagination.QueryOptions{
			TenantOpts: pagination.TenantOptions{
				BuID:   authCtx.BusinessUnitID,
				OrgID:  authCtx.OrganizationID,
				UserID: authCtx.UserID,
			},
		},
		StartDate: helpers.QueryInt64(c, "startDate"),
		EndDate:   helpers.QueryInt64(c, "endDate"),
		Type:      helpers.QueryString(c, "type"),
	}

	if err := c.ShouldBindQuery(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	ptoData, err := h.service.GetPTOCalendarData(c.Request.Context(), req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ptoData)
}

func (h *WorkerHandler) createPTO(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	req := new(worker.WorkerPTO)
	if err := c.ShouldBindJSON(req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, req)

	req, err := h.service.CreateWorkerPTO(c.Request.Context(), req, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, req)
}
