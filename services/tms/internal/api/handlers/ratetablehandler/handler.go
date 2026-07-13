package ratetablehandler

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratetable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ratetableservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ratetableservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratetableservice.Service
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
	resource := permission.ResourceRateTable.String()

	api := rg.Group("/rate-tables")
	api.GET(
		"/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.list,
	)
	api.GET(
		"/:rateTableID/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(resource, permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:rateTableID/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.update,
	)
	api.DELETE(
		"/:rateTableID/",
		h.pm.RequirePermission(resource, permission.OpDelete),
		h.delete,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:rateTableID/", h.getOption)
}

// @Summary List rate tables
// @ID listRateTables
// @Tags Rate Tables
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param lookupType query string false "Filter by lookup type"
// @Param active query bool false "Filter by active state"
// @Success 200 {object} pagination.Response[[]ratetable.RateTable]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	var active *bool
	if raw := helpers.QueryString(c, "active"); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			active = &parsed
		}
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratetable.RateTable], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRateTablesRequest{
					Filter:     req,
					LookupType: helpers.QueryString(c, "lookupType"),
					Active:     active,
				},
			)
		},
	)
}

// @Summary Get a rate table
// @ID getRateTable
// @Tags Rate Tables
// @Produce json
// @Param rateTableID path string true "Rate table ID"
// @Success 200 {object} ratetable.RateTable
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/{rateTableID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateTableID, err := pulid.MustParse(c.Param("rateTableID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetRateTableByIDRequest{
			RateTableID: rateTableID,
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

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a rate table
// @ID createRateTable
// @Tags Rate Tables
// @Accept json
// @Produce json
// @Param request body ratetable.RateTable true "Rate table payload"
// @Success 201 {object} ratetable.RateTable
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(ratetable.RateTable)
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

// @Summary Update a rate table
// @ID updateRateTable
// @Tags Rate Tables
// @Accept json
// @Produce json
// @Param rateTableID path string true "Rate table ID"
// @Param request body ratetable.RateTable true "Rate table payload"
// @Success 200 {object} ratetable.RateTable
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/{rateTableID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateTableID, err := pulid.MustParse(c.Param("rateTableID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(ratetable.RateTable)
	entity.ID = rateTableID
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

// @Summary Delete a rate table
// @ID deleteRateTable
// @Tags Rate Tables
// @Produce json
// @Param rateTableID path string true "Rate table ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/{rateTableID} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateTableID, err := pulid.MustParse(c.Param("rateTableID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), &repositories.GetRateTableByIDRequest{
		RateTableID: rateTableID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List rate table options
// @ID listRateTableOptions
// @Tags Rate Tables
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratetable.RateTable]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratetable.RateTable], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.RateTableSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a rate table option
// @ID getRateTableOption
// @Tags Rate Tables
// @Produce json
// @Param rateTableID path string true "Rate table ID"
// @Success 200 {object} ratetable.RateTable
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-tables/select-options/{rateTableID} [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	rateTableID, err := pulid.MustParse(c.Param("rateTableID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		&repositories.GetRateTableByIDRequest{
			RateTableID: rateTableID,
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
