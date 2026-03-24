package locationcategoryhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/locationcategory"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/locationcategoryservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *locationcategoryservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *locationcategoryservice.Service
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
	api := rg.Group("/location-categories")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceLocationCategory.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.GET(
		"/:locationCategoryID/",
		h.pm.RequirePermission(
			permission.ResourceLocationCategory.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceLocationCategory.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.PUT(
		"/:locationCategoryID/",
		h.pm.RequirePermission(
			permission.ResourceLocationCategory.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PATCH(
		"/:locationCategoryID/",
		h.pm.RequirePermission(
			permission.ResourceLocationCategory.String(),
			permission.OpUpdate,
		),
		h.patch,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:locationCategoryID", h.getOption)
}

// @Summary List location categories
// @ID listLocationCategories
// @Tags Location Categories
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]locationcategory.LocationCategory]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*locationcategory.LocationCategory], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListLocationCategoriesRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a location category option
// @ID getLocationCategoryOption
// @Tags Location Categories
// @Produce json
// @Param locationCategoryID path string true "Location category ID"
// @Success 200 {object} locationcategory.LocationCategory
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/select-options/{locationCategoryID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	locationCategoryID, err := pulid.MustParse(c.Param("locationCategoryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetLocationCategoryByIDRequest{
			ID: locationCategoryID,
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

// @Summary List location category options
// @ID listLocationCategoryOptions
// @Tags Location Categories
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]locationcategory.LocationCategory]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*locationcategory.LocationCategory], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get a location category
// @ID getLocationCategory
// @Tags Location Categories
// @Produce json
// @Param locationCategoryID path string true "Location category ID"
// @Success 200 {object} locationcategory.LocationCategory
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/{locationCategoryID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	locationCategoryID, err := pulid.MustParse(c.Param("locationCategoryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetLocationCategoryByIDRequest{
			ID: locationCategoryID,
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

// @Summary Create a location category
// @ID createLocationCategory
// @Tags Location Categories
// @Accept json
// @Produce json
// @Param request body locationcategory.LocationCategory true "Location category payload"
// @Success 201 {object} locationcategory.LocationCategory
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(locationcategory.LocationCategory)
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

// @Summary Update a location category
// @ID updateLocationCategory
// @Tags Location Categories
// @Accept json
// @Produce json
// @Param locationCategoryID path string true "Location category ID"
// @Param request body locationcategory.LocationCategory true "Location category payload"
// @Success 200 {object} locationcategory.LocationCategory
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/{locationCategoryID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	locationCategoryID, err := pulid.MustParse(c.Param("locationCategoryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(locationcategory.LocationCategory)
	entity.ID = locationCategoryID
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

// @Summary Patch a location category
// @ID patchLocationCategory
// @Tags Location Categories
// @Accept json
// @Produce json
// @Param locationCategoryID path string true "Location category ID"
// @Param request body locationcategory.LocationCategory true "Partial location category payload"
// @Success 200 {object} locationcategory.LocationCategory
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /location-categories/{locationCategoryID} [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	locationCategoryID, err := pulid.MustParse(c.Param("locationCategoryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetLocationCategoryByIDRequest{
			ID: locationCategoryID,
			TenantInfo: pagination.TenantInfo{
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

	actor := actorutil.FromAuthContext(authCtx)
	updatedEntity, err := h.service.Update(c.Request.Context(), existing, actor)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}
