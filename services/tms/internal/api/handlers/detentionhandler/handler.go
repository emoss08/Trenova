package detentionhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *detentionservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *detentionservice.Service
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
	resource := permission.ResourceDetentionPolicy.String()

	api := rg.Group("/detention")
	api.GET("/desk/", h.pm.RequirePermission(resource, permission.OpRead), h.desk)
	api.GET("/occurrences/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET(
		"/occurrences/:occurrenceID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.get,
	)
	api.GET(
		"/occurrences/:occurrenceID/dispute-packet/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.disputePacket,
	)
	api.POST(
		"/occurrences/:occurrenceID/waive/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.waive,
	)
	api.POST(
		"/occurrences/:occurrenceID/approve/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.approve,
	)
	api.POST("/backtest/", h.pm.RequirePermission(resource, permission.OpRead), h.backtest)
	api.GET("/facilities/", h.pm.RequirePermission(resource, permission.OpRead), h.facilities)
	api.GET("/customers/", h.pm.RequirePermission(resource, permission.OpRead), h.customers)
	api.GET("/waivers/", h.pm.RequirePermission(resource, permission.OpRead), h.waivers)

	api.POST(
		"/occurrences/:occurrenceID/dispute/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.dispute,
	)
}

func (h *Handler) tenant(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// @Summary List the live detention desk
// @ID listDetentionDesk
// @Tags Detention
// @Produce json
// @Success 200 {array} detentionservice.DeskEntry
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/desk/ [get]
func (h *Handler) desk(c *gin.Context) {
	entries, err := h.service.ListDesk(c.Request.Context(), h.tenant(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entries)
}

// @Summary List detention occurrences
// @ID listDetentionOccurrences
// @Tags Detention
// @Produce json
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param status query string false "Filter by status"
// @Param shipmentId query string false "Filter by shipment"
// @Param openOnly query bool false "Only still-accruing occurrences"
// @Success 200 {object} pagination.Response[[]detention.DetentionOccurrence]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	var shipmentID *pulid.ID
	if raw := helpers.QueryString(c, "shipmentId"); raw != "" {
		if parsed, err := pulid.MustParse(raw); err == nil {
			shipmentID = &parsed
		}
	}

	var customerID *pulid.ID
	if raw := helpers.QueryString(c, "customerId"); raw != "" {
		if parsed, err := pulid.MustParse(raw); err == nil {
			customerID = &parsed
		}
	}

	var locationID *pulid.ID
	if raw := helpers.QueryString(c, "locationId"); raw != "" {
		if parsed, err := pulid.MustParse(raw); err == nil {
			locationID = &parsed
		}
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*detention.DetentionOccurrence], error) {
			return h.service.ListOccurrences(
				c.Request.Context(),
				&repositories.ListDetentionOccurrencesRequest{
					Filter:     req,
					ShipmentID: shipmentID,
					CustomerID: customerID,
					LocationID: locationID,
					Status:     helpers.QueryString(c, "status"),
					OpenOnly:   helpers.QueryString(c, "openOnly") == "true",
				},
			)
		},
	)
}

// @Summary Get a detention occurrence with its evidence and defensibility score
// @ID getDetentionOccurrence
// @Tags Detention
// @Produce json
// @Param occurrenceID path string true "Occurrence ID"
// @Success 200 {object} detentionservice.OccurrenceDetail
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/{occurrenceID} [get]
func (h *Handler) get(c *gin.Context) {
	occurrenceID, err := pulid.MustParse(c.Param("occurrenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	detail, err := h.service.GetOccurrenceDetail(
		c.Request.Context(),
		&repositories.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: occurrenceID,
			TenantInfo:   h.tenant(c),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, detail)
}

// @Summary Build the dispute packet for an occurrence
// @ID getDetentionDisputePacket
// @Tags Detention
// @Produce json
// @Param occurrenceID path string true "Occurrence ID"
// @Success 200 {object} detentionservice.DisputePacket
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/{occurrenceID}/dispute-packet [get]
func (h *Handler) disputePacket(c *gin.Context) {
	occurrenceID, err := pulid.MustParse(c.Param("occurrenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	packet, err := h.service.BuildDisputePacket(
		c.Request.Context(), occurrenceID, h.tenant(c),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, packet)
}

type waiveRequest struct {
	Reason detention.WaiverReason `json:"reason" binding:"required"`
	Note   string                 `json:"note"`
}

// @Summary Waive a detention charge
// @ID waiveDetentionOccurrence
// @Tags Detention
// @Accept json
// @Produce json
// @Param occurrenceID path string true "Occurrence ID"
// @Param request body waiveRequest true "Waiver payload"
// @Success 200 {object} detention.DetentionOccurrence
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/{occurrenceID}/waive [post]
func (h *Handler) waive(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	occurrenceID, err := pulid.MustParse(c.Param("occurrenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(waiveRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	occurrence, err := h.service.Waive(c.Request.Context(), detentionservice.WaiveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   h.tenant(c),
		Reason:       req.Reason,
		Note:         req.Note,
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, occurrence)
}

// @Summary Approve a detention charge for billing
// @ID approveDetentionOccurrence
// @Tags Detention
// @Produce json
// @Param occurrenceID path string true "Occurrence ID"
// @Success 200 {object} detention.DetentionOccurrence
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/{occurrenceID}/approve [post]
func (h *Handler) approve(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	occurrenceID, err := pulid.MustParse(c.Param("occurrenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	occurrence, err := h.service.Approve(c.Request.Context(), detentionservice.ApproveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   h.tenant(c),
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, occurrence)
}

type disputeRequest struct {
	Note string `json:"note" binding:"required"`
}

// @Summary Record a customer dispute against a detention charge
// @ID disputeDetentionOccurrence
// @Tags Detention
// @Accept json
// @Produce json
// @Param occurrenceID path string true "Occurrence ID"
// @Param request body disputeRequest true "Dispute payload"
// @Success 200 {object} detention.DetentionOccurrence
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/occurrences/{occurrenceID}/dispute [post]
func (h *Handler) dispute(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	occurrenceID, err := pulid.MustParse(c.Param("occurrenceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req := new(disputeRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	occurrence, err := h.service.Dispute(c.Request.Context(), detentionservice.DisputeParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   h.tenant(c),
		Note:         req.Note,
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, occurrence)
}

func (h *Handler) statsRequest(c *gin.Context) *repositories.DetentionStatsRequest {
	req := &repositories.DetentionStatsRequest{TenantInfo: h.tenant(c)}

	if raw := helpers.QueryString(c, "from"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			req.From = parsed
		}
	}
	if raw := helpers.QueryString(c, "to"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			req.To = parsed
		}
	}
	if raw := helpers.QueryString(c, "limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			req.Limit = parsed
		}
	}

	return req
}

// @Summary Backtest a detention policy against settled history
// @ID backtestDetentionPolicy
// @Tags Detention
// @Accept json
// @Produce json
// @Param request body detentionservice.BacktestRequest true "Backtest request"
// @Success 200 {object} detentionservice.BacktestResult
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/backtest [post]
func (h *Handler) backtest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(detentionservice.BacktestRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if req.Policy != nil {
		authctx.AddContextToRequest(authCtx, req.Policy)
	}

	result, err := h.service.Backtest(c.Request.Context(), *req, h.tenant(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Detention profile per facility
// @ID listDetentionFacilityProfiles
// @Tags Detention
// @Produce json
// @Param from query int false "Window start (unix seconds)"
// @Param to query int false "Window end (unix seconds)"
// @Success 200 {array} repositories.FacilityDetentionStat
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/facilities/ [get]
func (h *Handler) facilities(c *gin.Context) {
	stats, err := h.service.FacilityProfiles(c.Request.Context(), h.statsRequest(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Detention margin per customer
// @ID listDetentionCustomerProfiles
// @Tags Detention
// @Produce json
// @Param from query int false "Window start (unix seconds)"
// @Param to query int false "Window end (unix seconds)"
// @Success 200 {array} repositories.CustomerDetentionStat
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/customers/ [get]
func (h *Handler) customers(c *gin.Context) {
	stats, err := h.service.CustomerProfiles(c.Request.Context(), h.statsRequest(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Detention waiver leakage by coded reason
// @ID listDetentionWaiverLeakage
// @Tags Detention
// @Produce json
// @Param from query int false "Window start (unix seconds)"
// @Param to query int false "Window end (unix seconds)"
// @Success 200 {array} repositories.WaiverLeakageStat
// @Failure 401 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/waivers/ [get]
func (h *Handler) waivers(c *gin.Context) {
	stats, err := h.service.WaiverLeakage(c.Request.Context(), h.statsRequest(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, stats)
}
