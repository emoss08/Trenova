package collectionhandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              collectionports.Service
	Reminders            paymentreminderports.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service   collectionports.Service
	reminders paymentreminderports.Service
	eh        *helpers.ErrorHandler
	pm        *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		service:   p.Service,
		reminders: p.Reminders,
		eh:        p.ErrorHandler,
		pm:        p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/collections")
	read := h.pm.RequirePermission(
		permission.ResourceAccountsReceivable.String(),
		permission.OpRead,
	)
	work := h.pm.RequirePermission(
		permission.ResourceAccountsReceivable.String(),
		permission.OpUpdate,
	)

	api.GET("/worklist/", read, h.worklist)
	api.GET("/customers/:customerID/", read, h.account)
	api.POST("/customers/:customerID/activities/", work, h.logActivity)
	api.PUT("/customers/:customerID/collector/", work, h.assignCollector)
	api.PUT("/customers/:customerID/follow-up/", work, h.scheduleFollowUp)
	api.POST("/customers/:customerID/reminder/", work, h.sendReminder)
}

func (h *Handler) sendReminder(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(paymentreminderports.SendNowRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)
	req.CustomerID = customerID

	sent, err := h.reminders.SendNow(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reminders": sent})
}

func tenantOf(c *gin.Context) pagination.TenantInfo {
	auth := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}
}

func (h *Handler) worklist(c *gin.Context) {
	req := new(collectionports.WorklistRequest)
	if err := c.ShouldBindQuery(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)

	page, err := h.service.Worklist(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (h *Handler) account(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	var before *collectionports.ActivityCursor
	if raw := c.Query("beforeCreatedAt"); raw != "" {
		createdAt, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			h.eh.HandleError(c, parseErr)
			return
		}
		beforeID, parseErr := pulid.MustParse(c.Query("beforeId"))
		if parseErr != nil {
			h.eh.HandleError(c, parseErr)
			return
		}
		before = &collectionports.ActivityCursor{CreatedAt: createdAt, ID: beforeID}
	}

	view, err := h.service.GetAccount(c.Request.Context(), collectionports.GetAccountRequest{
		TenantInfo: tenantOf(c),
		CustomerID: customerID,
	}, before)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) logActivity(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(collectionports.LogActivityRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)
	req.CustomerID = customerID

	view, err := h.service.LogActivity(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, view)
}

func (h *Handler) assignCollector(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(collectionports.AssignCollectorRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)
	req.CustomerID = customerID

	view, err := h.service.AssignCollector(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}

func (h *Handler) scheduleFollowUp(c *gin.Context) {
	customerID, err := pulid.MustParse(c.Param("customerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req := new(collectionports.ScheduleFollowUpRequest)
	if err = c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = tenantOf(c)
	req.CustomerID = customerID

	view, err := h.service.ScheduleFollowUp(
		c.Request.Context(),
		req,
		actorutil.FromAuthContext(authctx.GetAuthContext(c)),
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, view)
}
