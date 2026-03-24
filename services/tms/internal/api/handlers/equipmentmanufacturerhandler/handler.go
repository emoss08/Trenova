package equipmentmanufacturerhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/equipmentmanufacturerservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *equipmentmanufacturerservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *equipmentmanufacturerservice.Service
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
	api := rg.Group("/equipment-manufacturers")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpRead,
		),
		h.list,
	)
	api.GET(
		"/:equipManufacturerID/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.PUT(
		"/:equipManufacturerID/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PATCH(
		"/:equipManufacturerID/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpUpdate,
		),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(
			permission.ResourceEquipmentManufacturer.String(),
			permission.OpUpdate,
		),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:equipManufacturerID", h.getOption)
}

// @Summary List equipment manufacturers
// @ID listEquipmentManufacturers
// @Tags Equipment Manufacturers
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]equipmentmanufacturer.EquipmentManufacturer]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListEquipmentManufacturersRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get an equipment manufacturer option
// @ID getEquipmentManufacturerOption
// @Tags Equipment Manufacturers
// @Produce json
// @Param equipManufacturerID path string true "Equipment manufacturer ID"
// @Success 200 {object} equipmentmanufacturer.EquipmentManufacturer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/select-options/{equipManufacturerID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	equipManufacturerID, err := pulid.MustParse(c.Param("equipManufacturerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentManufacturerByIDRequest{
			ID: equipManufacturerID,
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

// @Summary List equipment manufacturer options
// @ID listEquipmentManufacturerOptions
// @Tags Equipment Manufacturers
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]equipmentmanufacturer.EquipmentManufacturer]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*equipmentmanufacturer.EquipmentManufacturer], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

// @Summary Get an equipment manufacturer
// @ID getEquipmentManufacturer
// @Tags Equipment Manufacturers
// @Produce json
// @Param equipManufacturerID path string true "Equipment manufacturer ID"
// @Success 200 {object} equipmentmanufacturer.EquipmentManufacturer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/{equipManufacturerID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	equipManufacturerID, err := pulid.MustParse(c.Param("equipManufacturerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentManufacturerByIDRequest{
			ID: equipManufacturerID,
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

// @Summary Create an equipment manufacturer
// @ID createEquipmentManufacturer
// @Tags Equipment Manufacturers
// @Accept json
// @Produce json
// @Param request body equipmentmanufacturer.EquipmentManufacturer true "Equipment manufacturer payload"
// @Success 201 {object} equipmentmanufacturer.EquipmentManufacturer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
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

// @Summary Update an equipment manufacturer
// @ID updateEquipmentManufacturer
// @Tags Equipment Manufacturers
// @Accept json
// @Produce json
// @Param equipManufacturerID path string true "Equipment manufacturer ID"
// @Param request body equipmentmanufacturer.EquipmentManufacturer true "Equipment manufacturer payload"
// @Success 200 {object} equipmentmanufacturer.EquipmentManufacturer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/{equipManufacturerID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	equipManufacturerID, err := pulid.MustParse(c.Param("equipManufacturerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(equipmentmanufacturer.EquipmentManufacturer)
	entity.ID = equipManufacturerID
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

// @Summary Patch an equipment manufacturer
// @ID patchEquipmentManufacturer
// @Tags Equipment Manufacturers
// @Accept json
// @Produce json
// @Param equipManufacturerID path string true "Equipment manufacturer ID"
// @Param request body equipmentmanufacturer.EquipmentManufacturer true "Partial equipment manufacturer payload"
// @Success 200 {object} equipmentmanufacturer.EquipmentManufacturer
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-manufacturers/{equipManufacturerID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	equipManufacturerID, err := pulid.MustParse(c.Param("equipManufacturerID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentManufacturerByIDRequest{
			ID: equipManufacturerID,
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

func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update equipment manufacturers"),
		)
		return
	}

	req := new(repositories.BulkUpdateEquipmentManufacturerStatusRequest)
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

	c.JSON(http.StatusOK, results)
}
