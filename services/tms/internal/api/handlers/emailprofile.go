package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type EmailProfileHandlerParams struct {
	fx.In

	Service      services.EmailProfileService
	EmailService services.EmailService
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type EmailProfileHandler struct {
	service      services.EmailProfileService
	emailService services.EmailService
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewEmailProfileHandler(p EmailProfileHandlerParams) *EmailProfileHandler {
	return &EmailProfileHandler{
		service:      p.Service,
		emailService: p.EmailService,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *EmailProfileHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/email-profiles/")
	api.GET("", h.pm.RequirePermission(permission.ResourceEmailProfile, "read"), h.list)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceEmailProfile, "read"), h.get)
	api.POST("", h.pm.RequirePermission(permission.ResourceEmailProfile, "create"), h.create)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceEmailProfile, "update"), h.update)
	api.POST(
		"test-connection/",
		h.pm.RequirePermission(permission.ResourceEmailProfile, "create"),
		h.testConnection,
	)
}

func (h *EmailProfileHandler) list(c *gin.Context) {
	pagination.Handle[*email.EmailProfile](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*email.EmailProfile], error) {
			return h.service.List(c.Request.Context(), &repositories.ListEmailProfileRequest{
				Filter:          opts,
				ExcludeInactive: helpers.QueryBool(c, "excludeInactive"),
			})
		})
}

func (h *EmailProfileHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEmailProfileByIDRequest{
			ProfileID:  id,
			OrgID:      authCtx.OrganizationID,
			BuID:       authCtx.BusinessUnitID,
			UserID:     authCtx.UserID,
			ExpandData: helpers.QueryBool(c, "expandData"),
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *EmailProfileHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(email.EmailProfile)
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

func (h *EmailProfileHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(email.EmailProfile)
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

func (h *EmailProfileHandler) testConnection(c *gin.Context) {
	var req services.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	success, err := h.emailService.TestConnection(c.Request.Context(), &req)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": success})
}
