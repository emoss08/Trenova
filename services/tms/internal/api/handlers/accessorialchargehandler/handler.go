package accessorialchargehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/accessorialchargeservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *accessorialchargeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *accessorialchargeservice.Service
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
	api := rg.Group("/accessorial-charges")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:accessorialChargeID/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:accessorialChargeID/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:accessorialChargeID/",
		h.pm.RequirePermission(permission.ResourceAccessorialCharge.String(), permission.OpUpdate),
		h.patch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:accessorialChargeID/", h.getOption)
}

// @Summary List accessorial charges
// @ID listAccessorialCharges
// @Tags Accessorial Charges
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]accessorialcharge.AccessorialCharge]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListAccessorialChargeRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get an accessorial charge option
// @ID getAccessorialChargeOption
// @Tags Accessorial Charges
// @Produce json
// @Param accessorialChargeID path string true "Accessorial charge ID"
// @Success 200 {object} accessorialcharge.AccessorialCharge
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/select-options/{accessorialChargeID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	accessorialChargeID, err := pulid.MustParse(c.Param("accessorialChargeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetAccessorialChargeByIDRequest{
			ID: accessorialChargeID,
			TenantInfo: &pagination.TenantInfo{
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

// @Summary List accessorial charge options
// @ID listAccessorialChargeOptions
// @Tags Accessorial Charges
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]accessorialcharge.AccessorialCharge]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*accessorialcharge.AccessorialCharge], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get an accessorial charge
// @ID getAccessorialCharge
// @Tags Accessorial Charges
// @Produce json
// @Param accessorialChargeID path string true "Accessorial charge ID"
// @Success 200 {object} accessorialcharge.AccessorialCharge
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/{accessorialChargeID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	accessorialChargeID, err := pulid.MustParse(c.Param("accessorialChargeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetAccessorialChargeByIDRequest{
			ID: accessorialChargeID,
			TenantInfo: &pagination.TenantInfo{
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

// @Summary Create an accessorial charge
// @ID createAccessorialCharge
// @Tags Accessorial Charges
// @Accept json
// @Produce json
// @Param request body accessorialcharge.AccessorialCharge true "Accessorial charge payload"
// @Success 201 {object} accessorialcharge.AccessorialCharge
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(accessorialcharge.AccessorialCharge)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update an accessorial charge
// @ID updateAccessorialCharge
// @Tags Accessorial Charges
// @Accept json
// @Produce json
// @Param accessorialChargeID path string true "Accessorial charge ID"
// @Param request body accessorialcharge.AccessorialCharge true "Accessorial charge payload"
// @Success 200 {object} accessorialcharge.AccessorialCharge
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/{accessorialChargeID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	accessorialChargeID, err := pulid.MustParse(c.Param("accessorialChargeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(accessorialcharge.AccessorialCharge)
	entity.ID = accessorialChargeID
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Patch an accessorial charge
// @ID patchAccessorialCharge
// @Tags Accessorial Charges
// @Accept json
// @Produce json
// @Param accessorialChargeID path string true "Accessorial charge ID"
// @Param request body accessorialcharge.AccessorialCharge true "Partial accessorial charge payload"
// @Success 200 {object} accessorialcharge.AccessorialCharge
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accessorial-charges/{accessorialChargeID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	accessorialChargeID, err := pulid.MustParse(c.Param("accessorialChargeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetAccessorialChargeByIDRequest{
			ID: accessorialChargeID,
			TenantInfo: &pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
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

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
