package servicetypehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/app/servicetypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/servicetypeports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *servicetypeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *servicetypeservice.Service
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
	api := rg.Group("/service-types")
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceServiceType.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:serviceTypeID/",
		h.pm.RequirePermission(permission.ResourceServiceType.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:serviceTypeID/",
		h.pm.RequirePermission(permission.ResourceServiceType.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceServiceType.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)
}

// @Summary Bulk update service type statuses
// @ID bulkUpdateServiceTypeStatus
// @Tags Service Types
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateServiceTypeStatusRequest true "Bulk status update request"
// @Success 200 {array} servicetype.ServiceType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /service-types/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update service types"),
		)
		return
	}

	req := new(servicetypeports.BulkUpdateServiceTypeStatusRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}

	results, err := h.service.BulkUpdateStatus(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, results)
}

// @Summary Create a service type
// @ID createServiceType
// @Tags Service Types
// @Accept json
// @Produce json
// @Param request body servicetype.ServiceType true "Service type payload"
// @Success 201 {object} servicetype.ServiceType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /service-types/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(servicetype.ServiceType)
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

	helpers.JSON(c, http.StatusCreated, created)
}

// @Summary Patch a service type
// @ID patchServiceType
// @Tags Service Types
// @Accept json
// @Produce json
// @Param serviceTypeID path string true "Service type ID"
// @Param request body servicetype.ServiceType true "Partial service type payload"
// @Success 200 {object} servicetype.ServiceType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /service-types/{serviceTypeID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	serviceTypeID, err := pulid.MustParse(c.Param("serviceTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		servicetypeports.GetServiceTypeByIDRequest{
			ID:         serviceTypeID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
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
	updatedEntity, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updatedEntity)
}

// @Summary Update a service type
// @ID updateServiceType
// @Tags Service Types
// @Accept json
// @Produce json
// @Param serviceTypeID path string true "Service type ID"
// @Param request body servicetype.ServiceType true "Service type payload"
// @Success 200 {object} servicetype.ServiceType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /service-types/{serviceTypeID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	serviceTypeID, err := pulid.MustParse(c.Param("serviceTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(servicetype.ServiceType)
	entity.ID = serviceTypeID
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

	helpers.JSON(c, http.StatusOK, updated)
}
