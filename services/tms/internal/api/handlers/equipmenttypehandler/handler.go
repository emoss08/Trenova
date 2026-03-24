package equipmenttypehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/equipmenttypeservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *equipmenttypeservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *equipmenttypeservice.Service
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
	api := rg.Group("/equipment-types")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:equipTypeID/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:equipTypeID/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:equipTypeID/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpUpdate),
		h.patch,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(permission.ResourceEquipmentType.String(), permission.OpUpdate),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:equipTypeID", h.getOption)
}

// @Summary List all equipment types
// @Description Returns paginated equipment types. Protected routes accept a Bearer token or an authenticated session cookie.
// @ID listEquipmentTypes
// @Tags Equipment Types
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param classes query []string false "Filter by equipment class" collectionFormat(multi) Enums(Tractor,Trailer,Container,Other)
// @Success 200 {object} pagination.Response[[]equipmenttype.EquipmentType]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListEquipmentTypesRequest{
					Filter:  req,
					Classes: helpers.QuerySlice(c, "classes", []string{}),
				},
			)
		},
	)
}

// @Summary Bulk update equipment type statuses
// @ID bulkUpdateEquipmentTypeStatus
// @Tags Equipment Types
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateEquipmentTypeStatusRequest true "Bulk status update request"
// @Success 200 {array} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.IsAPIKey() {
		h.eh.HandleError(
			c,
			errortypes.NewAuthorizationError("API keys cannot bulk update equipment types"),
		)
		return
	}

	req := new(repositories.BulkUpdateEquipmentTypeStatusRequest)
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

// @Summary Get a selectable equipment type option
// @ID getEquipmentTypeOption
// @Tags Equipment Types
// @Produce json
// @Param equipTypeID path string true "Equipment type ID"
// @Success 200 {object} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/select-options/{equipTypeID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	equipTypeID, err := pulid.MustParse(c.Param("equipTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetEquipmentTypeByIDRequest{
		ID: equipTypeID,
		TenantInfo: pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary List equipment type select options
// @ID listEquipmentTypeOptions
// @Tags Equipment Types
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param classes query []string false "Filter by equipment class" collectionFormat(multi) Enums(Tractor,Trailer,Container,Other)
// @Success 200 {object} pagination.Response[[]equipmenttype.EquipmentType]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*equipmenttype.EquipmentType], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.EquipmentTypeSelectOptionsRequest{
					SelectQueryRequest: req,
					Classes:            helpers.QuerySlice(c, "classes", []string{}),
				},
			)
		},
	)
}

// @Summary Get an equipment type
// @ID getEquipmentType
// @Tags Equipment Types
// @Produce json
// @Param equipTypeID path string true "Equipment type ID"
// @Success 200 {object} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/{equipTypeID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	equipTypeID, err := pulid.MustParse(c.Param("equipTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentTypeByIDRequest{
			ID: equipTypeID,
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

// @Summary Create an equipment type
// @ID createEquipmentType
// @Tags Equipment Types
// @Accept json
// @Produce json
// @Param request body equipmenttype.EquipmentType true "Equipment type payload"
// @Success 201 {object} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(equipmenttype.EquipmentType)
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

// @Summary Patch an equipment type
// @ID patchEquipmentType
// @Tags Equipment Types
// @Accept json
// @Produce json
// @Param equipTypeID path string true "Equipment type ID"
// @Param request body equipmenttype.EquipmentType true "Partial equipment type payload"
// @Success 200 {object} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/{equipTypeID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	equipTypeID, err := pulid.MustParse(c.Param("equipTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetEquipmentTypeByIDRequest{
			ID: equipTypeID,
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

// @Summary Update an equipment type
// @ID updateEquipmentType
// @Tags Equipment Types
// @Accept json
// @Produce json
// @Param equipTypeID path string true "Equipment type ID"
// @Param request body equipmenttype.EquipmentType true "Equipment type payload"
// @Success 200 {object} equipmenttype.EquipmentType
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /equipment-types/{equipTypeID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	equipTypeID, err := pulid.MustParse(c.Param("equipTypeID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(equipmenttype.EquipmentType)
	entity.ID = equipTypeID
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
