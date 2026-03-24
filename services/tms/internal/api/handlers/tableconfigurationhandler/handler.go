package tableconfigurationhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tableconfigurationservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      *tableconfigurationservice.Service
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service *tableconfigurationservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/table-configurations")
	api.GET("/", h.list)
	api.GET("/default", h.getDefault)
	api.GET("/:id", h.get)
	api.POST("/", h.create)
	api.PUT("/:id", h.update)
	api.PATCH("/:id", h.patch)
	api.DELETE("/:id", h.delete)
	api.POST("/:id/set-default", h.setDefault)
}

// @Summary List table configurations
// @ID listTableConfigurations
// @Tags Table Configurations
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param resource query string false "Filter by resource"
// @Param visibility query string false "Filter by visibility"
// @Success 200 {object} pagination.Response[[]tableconfiguration.TableConfiguration]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
			return h.service.List(c.Request.Context(), &repositories.ListTableConfigurationsRequest{
				Filter:     req,
				Resource:   helpers.QueryString(c, "resource"),
				Visibility: helpers.QueryString(c, "visibility"),
			})
		},
	)
}

// @Summary Get a table configuration
// @ID getTableConfiguration
// @Tags Table Configurations
// @Produce json
// @Param id path string true "Table configuration ID"
// @Success 200 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/{id} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetTableConfigurationByIDRequest{
			ConfigurationID: id,
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

// @Summary Get the default table configuration
// @ID getDefaultTableConfiguration
// @Tags Table Configurations
// @Produce json
// @Param resource query string true "Resource name"
// @Success 200 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} gin.H
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/default [get]
func (h *Handler) getDefault(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	resource := helpers.QueryString(c, "resource")

	if resource == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource query parameter is required"})
		return
	}

	entity, err := h.service.GetDefaultForResource(
		c.Request.Context(),
		repositories.GetDefaultTableConfigurationRequest{
			Resource: resource,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			c.JSON(http.StatusOK, nil)
			return
		}

		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a table configuration
// @ID createTableConfiguration
// @Tags Table Configurations
// @Accept json
// @Produce json
// @Param request body tableconfiguration.TableConfiguration true "Table configuration payload"
// @Success 201 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(tableconfiguration.TableConfiguration)
	authctx.AddContextToRequest(authCtx, entity)

	if err := c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// @Summary Update a table configuration
// @ID updateTableConfiguration
// @Tags Table Configurations
// @Accept json
// @Produce json
// @Param id path string true "Table configuration ID"
// @Param request body tableconfiguration.TableConfiguration true "Table configuration payload"
// @Success 200 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/{id} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(tableconfiguration.TableConfiguration)
	authctx.AddContextToRequest(authCtx, entity)
	entity.ID = id

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Patch a table configuration
// @ID patchTableConfiguration
// @Tags Table Configurations
// @Accept json
// @Produce json
// @Param id path string true "Table configuration ID"
// @Param request body tableconfiguration.TableConfiguration true "Table configuration payload"
// @Success 200 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/{id} [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetTableConfigurationByIDRequest{
			ConfigurationID: id,
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

	updated, err := h.service.Update(c.Request.Context(), existing)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a table configuration
// @ID deleteTableConfiguration
// @Tags Table Configurations
// @Param id path string true "Table configuration ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/{id} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), id, pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Set the default table configuration
// @ID setDefaultTableConfiguration
// @Tags Table Configurations
// @Produce json
// @Param id path string true "Table configuration ID"
// @Success 200 {object} tableconfiguration.TableConfiguration
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /table-configurations/{id}/set-default [post]
func (h *Handler) setDefault(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.SetDefault(c.Request.Context(), id, pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}
