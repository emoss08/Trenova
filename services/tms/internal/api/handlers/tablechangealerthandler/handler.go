package tablechangealerthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tablechangealert"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tablechangealertservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *tablechangealertservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *tablechangealertservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	tca := rg.Group("/tca")

	allowlist := tca.Group("/allowlisted-tables")
	allowlist.GET("/", h.listAllowlistedTables)

	subs := tca.Group("/subscriptions")
	subs.GET("/", h.listSubscriptions)
	subs.GET("/:id", h.getSubscription)
	subs.POST("/", h.createSubscription)
	subs.PUT("/:id", h.updateSubscription)
	subs.DELETE("/:id", h.deleteSubscription)
	subs.PATCH("/:id/pause", h.pauseSubscription)
	subs.PATCH("/:id/resume", h.resumeSubscription)

}

func (h *Handler) listAllowlistedTables(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	tables, err := h.service.ListAllowlistedTables(c.Request.Context(), pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, tables)
}

func (h *Handler) listSubscriptions(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, auth)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tablechangealert.TCASubscription], error) {
			return h.service.ListSubscriptions(c.Request.Context(), &repositories.ListTCASubscriptionsRequest{
				Filter: req,
			})
		},
	)
}

func (h *Handler) getSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetSubscriptionByID(
		c.Request.Context(),
		repositories.GetTCASubscriptionByIDRequest{
			SubscriptionID: id,
			TenantInfo: pagination.TenantInfo{
				OrgID:  auth.OrganizationID,
				BuID:   auth.BusinessUnitID,
				UserID: auth.UserID,
			},
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *Handler) createSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	entity := new(tablechangealert.TCASubscription)
	authctx.AddContextToRequest(auth, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.CreateSubscription(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handler) updateSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tablechangealert.TCASubscription)
	authctx.AddContextToRequest(auth, entity)
	entity.ID = id

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.UpdateSubscription(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) deleteSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.DeleteSubscription(c.Request.Context(), id, pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) pauseSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.PauseSubscription(c.Request.Context(), id, pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *Handler) resumeSubscription(c *gin.Context) {
	auth := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.ResumeSubscription(c.Request.Context(), id, pagination.TenantInfo{
		OrgID:  auth.OrganizationID,
		BuID:   auth.BusinessUnitID,
		UserID: auth.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
