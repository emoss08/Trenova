package workerptohandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerptoservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *workerptoservice.Service
	PermissionMiddleware *middleware.PermissionMiddleware
	ErrorHandler         *helpers.ErrorHandler
}

type Handler struct {
	service *workerptoservice.Service
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
	api := rg.Group("/worker-pto")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/upcoming/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpRead),
		h.listUpcoming,
	)
	api.GET(
		"/:ptoID/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpCreate),
		h.create,
	)
	api.POST(
		"/:ptoID/approve/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpApprove),
		h.approve,
	)
	api.POST(
		"/:ptoID/reject/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpReject),
		h.reject,
	)
	api.GET(
		"/chart/",
		h.pm.RequirePermission(permission.ResourceWorkerPTO.String(), permission.OpRead),
		h.chartData,
	)
}

// @Summary List worker PTO entries
// @ID listWorkerPTO
// @Tags Worker PTO
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param status query string false "Filter by PTO status"
// @Param type query string false "Filter by PTO type"
// @Param startDateFrom query int false "Start date from"
// @Param startDateTo query int false "Start date to"
// @Param workerId query string false "Worker ID"
// @Param includeWorker query bool false "Include worker details"
// @Success 200 {object} pagination.Response[[]worker.WorkerPTO]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*worker.WorkerPTO], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListPTORequest{
					Filter:        req,
					Status:        helpers.QueryString(c, "status", ""),
					Type:          helpers.QueryString(c, "type", ""),
					StartDateFrom: helpers.QueryInt64(c, "startDateFrom", 0),
					StartDateTo:   helpers.QueryInt64(c, "startDateTo", 0),
					WorkerID:      helpers.QueryPulid(c, "workerId"),
					IncludeWorker: helpers.QueryBool(c, "includeWorker", false),
				},
			)
		},
	)
}

// @Summary List upcoming worker PTO entries
// @ID listUpcomingWorkerPTO
// @Tags Worker PTO
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param status query string false "Filter by PTO status"
// @Param type query string false "Filter by PTO type"
// @Param startDate query int false "Start date"
// @Param endDate query int false "End date"
// @Param workerId query string false "Worker ID"
// @Param fleetCodeId query string false "Fleet code ID"
// @Param timezone query string false "Timezone"
// @Success 200 {object} pagination.Response[[]worker.WorkerPTO]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/upcoming/ [get]
func (h *Handler) listUpcoming(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*worker.WorkerPTO], error) {
			return h.service.ListUpcoming(
				c.Request.Context(),
				&repositories.ListUpcomingPTORequest{
					Filter: req,
					ListWorkerPTOFilterOptions: repositories.ListWorkerPTOFilterOptions{
						Status:      helpers.QueryString(c, "status", ""),
						Type:        helpers.QueryString(c, "type", ""),
						StartDate:   helpers.QueryInt64(c, "startDate", 0),
						EndDate:     helpers.QueryInt64(c, "endDate", 0),
						WorkerID:    helpers.QueryString(c, "workerId"),
						FleetCodeID: helpers.QueryString(c, "fleetCodeId"),
						Timezone:    helpers.QueryString(c, "timezone", "UTC"),
					},
				},
			)
		},
	)
}

// @Summary Get a worker PTO entry
// @ID getWorkerPTO
// @Tags Worker PTO
// @Produce json
// @Param ptoID path string true "Worker PTO ID"
// @Param includeWorker query bool false "Include worker details"
// @Success 200 {object} worker.WorkerPTO
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/{ptoID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ptoID, err := pulid.MustParse(c.Param("ptoID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetPTOByIDRequest{
			ID: ptoID,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
			IncludeWorker: helpers.QueryBool(c, "includeWorker", false),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(worker.WorkerPTO)
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}
}

// @Summary Approve a worker PTO entry
// @ID approveWorkerPTO
// @Tags Worker PTO
// @Produce json
// @Param ptoID path string true "Worker PTO ID"
// @Success 200 {object} worker.WorkerPTO
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/{ptoID}/approve/ [post]
func (h *Handler) approve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ptoID, err := pulid.MustParse(c.Param("ptoID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Approve(
		c.Request.Context(),
		&repositories.UpdatePTOStatusRequest{
			ID:     ptoID,
			Status: worker.PTOStatusApproved,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Reject a worker PTO entry
// @ID rejectWorkerPTO
// @Tags Worker PTO
// @Produce json
// @Param ptoID path string true "Worker PTO ID"
// @Success 200 {object} worker.WorkerPTO
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/{ptoID}/reject/ [post]
func (h *Handler) reject(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ptoID, err := pulid.MustParse(c.Param("ptoID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Reject(
		c.Request.Context(),
		&repositories.UpdatePTOStatusRequest{
			ID:     ptoID,
			Status: worker.PTOStatusRejected,
			TenantInfo: pagination.TenantInfo{
				OrgID: authCtx.OrganizationID,
				BuID:  authCtx.BusinessUnitID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Get worker PTO chart data
// @ID getWorkerPTOChartData
// @Tags Worker PTO
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param startDateFrom query int false "Start date from"
// @Param startDateTo query int false "Start date to"
// @Param type query string false "PTO type"
// @Param workerId query string false "Worker ID"
// @Param timezone query string false "Timezone"
// @Success 200 {array} repositories.PTOChartDataPoint
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /worker-pto/chart/ [get]
func (h *Handler) chartData(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	data, err := h.service.GetChartData(
		c.Request.Context(),
		&repositories.PTOChartRequest{
			Filter:        req,
			StartDateFrom: helpers.QueryInt64(c, "startDateFrom"),
			StartDateTo:   helpers.QueryInt64(c, "startDateTo"),
			Type:          helpers.QueryString(c, "type"),
			WorkerID:      helpers.QueryString(c, "workerId"),
			Timezone:      helpers.QueryString(c, "timezone", "UTC"),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, data)
}
