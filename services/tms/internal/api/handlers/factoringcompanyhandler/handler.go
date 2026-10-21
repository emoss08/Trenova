package factoringcompanyhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/responseguard"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/app/factoringcompanyservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *factoringcompanyservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *factoringcompanyservice.Service
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
	api := rg.Group("/factoring-companies")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceFactoringCompany.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:factoringCompanyID",
		h.pm.RequirePermission(permission.ResourceFactoringCompany.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceFactoringCompany.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:factoringCompanyID/",
		h.pm.RequirePermission(permission.ResourceFactoringCompany.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:factoringCompanyID/",
		h.pm.RequirePermission(permission.ResourceFactoringCompany.String(), permission.OpUpdate),
		h.patch,
	)

	selectOptions := api.Group("/select-options", responseguard.References())
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:factoringCompanyID/", h.getOption)
}

// @Summary List factoring companies
// @ID listFactoringCompanys
// @Tags Factoring Companies
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]factoringcompany.FactoringCompany]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			return h.service.List(
				c.Request.Context(),
				&factoringcompanyports.ListFactoringCompanyRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a factoring company
// @ID getFactoringCompany
// @Tags Factoring Companies
// @Produce json
// @Param factoringCompanyID path string true "Factoring company ID"
// @Success 200 {object} factoringcompany.FactoringCompany
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/{factoringCompanyID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	factoringCompanyID, err := pulid.MustParse(c.Param("factoringCompanyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		factoringcompanyports.GetFactoringCompanyByIDRequest{
			ID:         factoringCompanyID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Create a factoring company
// @ID createFactoringCompany
// @Tags Factoring Companies
// @Accept json
// @Produce json
// @Param request body factoringcompany.FactoringCompany true "Factoring company payload"
// @Success 201 {object} factoringcompany.FactoringCompany
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(factoringcompany.FactoringCompany)
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	created, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusCreated, created)
}

// @Summary Update a factoring company
// @ID updateFactoringCompany
// @Tags Factoring Companies
// @Accept json
// @Produce json
// @Param factoringCompanyID path string true "Factoring company ID"
// @Param request body factoringcompany.FactoringCompany true "Factoring company payload"
// @Success 200 {object} factoringcompany.FactoringCompany
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/{factoringCompanyID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	factoringCompanyID, err := pulid.MustParse(c.Param("factoringCompanyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(factoringcompany.FactoringCompany)
	entity.ID = factoringCompanyID
	if err = authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updated)
}

// @Summary Patch a factoring company
// @ID patchFactoringCompany
// @Tags Factoring Companies
// @Accept json
// @Produce json
// @Param factoringCompanyID path string true "Factoring company ID"
// @Param request body factoringcompany.FactoringCompany true "Partial factoring company payload"
// @Success 200 {object} factoringcompany.FactoringCompany
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/{factoringCompanyID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	factoringCompanyID, err := pulid.MustParse(c.Param("factoringCompanyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		factoringcompanyports.GetFactoringCompanyByIDRequest{
			ID:         factoringCompanyID,
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

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updatedEntity)
}

// @Summary List factoring company options
// @ID listFactoringCompanyOptions
// @Tags Factoring Companies
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]factoringcompany.FactoringCompany]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*factoringcompany.FactoringCompany], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&factoringcompanyports.FactoringCompanySelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a factoring company option
// @ID getFactoringCompanyOption
// @Tags Factoring Companies
// @Produce json
// @Param factoringCompanyID path string true "Factoring company ID"
// @Success 200 {object} factoringcompany.FactoringCompany
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /factoring-companies/select-options/{factoringCompanyID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	factoringCompanyID, err := pulid.MustParse(c.Param("factoringCompanyID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		factoringcompanyports.GetFactoringCompanyByIDRequest{
			ID:         factoringCompanyID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}
