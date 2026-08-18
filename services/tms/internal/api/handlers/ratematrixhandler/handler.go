package ratematrixhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ratematrixservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *ratematrixservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *ratematrixservice.Service
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
	resource := permission.ResourceRateMatrix.String()

	api := rg.Group("/rate-matrices")
	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.GET("/:rateMatrixID/", h.pm.RequirePermission(resource, permission.OpRead), h.get)
	api.POST("/", h.pm.RequirePermission(resource, permission.OpCreate), h.create)
	api.PUT("/:rateMatrixID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.update)
	api.DELETE("/:rateMatrixID/", h.pm.RequirePermission(resource, permission.OpDelete), h.delete)

	api.GET(
		"/:rateMatrixID/cells/",
		h.pm.RequirePermission(resource, permission.OpRead),
		h.listCells,
	)
	api.PUT(
		"/:rateMatrixID/cells/",
		h.pm.RequirePermission(resource, permission.OpUpdate),
		h.replaceCells,
	)

	density := api.Group("/density-scales")
	density.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.listDensityScales)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
}

// @Summary List rate matrices
// @ID listRateMatrices
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrix]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListRateMatrixRequest{Filter: req},
			)
		},
	)
}

// @Summary Get a rate matrix
// @ID getRateMatrix
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), &repositories.GetRateMatrixByIDRequest{
		RateMatrixID:      matrixID,
		TenantInfo:        tenantOf(authCtx),
		IncludeDimensions: true,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Create a rate matrix
// @ID createRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param request body ratematrix.RateMatrix true "Rate matrix payload"
// @Success 201 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(ratematrix.RateMatrix)
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

// @Summary Update a rate matrix
// @ID updateRateMatrix
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body ratematrix.RateMatrix true "Rate matrix payload"
// @Success 200 {object} ratematrix.RateMatrix
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	entity := new(ratematrix.RateMatrix)
	authctx.AddContextToRequest(authCtx, entity)

	if err = c.ShouldBindJSON(entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity.ID = matrixID

	updated, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// @Summary Delete a rate matrix
// @ID deleteRateMatrix
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 404 {object} helpers.ProblemDetail
// @Failure 409 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID} [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	if err = h.service.Delete(c.Request.Context(), &repositories.GetRateMatrixByIDRequest{
		RateMatrixID: matrixID,
		TenantInfo:   tenantOf(authCtx),
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List a rate matrix's cells
// @ID listRateMatrixCells
// @Tags Rate Matrices
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrixCell]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/cells [get]
func (h *Handler) listCells(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
			return h.service.ListCells(
				c.Request.Context(),
				&repositories.ListRateMatrixCellsRequest{
					TenantInfo:   tenantOf(authCtx),
					RateMatrixID: matrixID,
					Filter:       req,
				},
			)
		},
	)
}

type replaceCellsRequest struct {
	Cells []*ratematrix.RateMatrixCell `json:"cells"`
}

// @Summary Replace a rate matrix's cells
// @Description Swaps the entire grid for the one supplied. A tariff arrives as a whole sheet, and merging would leave behind whatever the new sheet dropped.
// @ID replaceRateMatrixCells
// @Tags Rate Matrices
// @Accept json
// @Produce json
// @Param rateMatrixID path string true "Rate matrix ID"
// @Param request body replaceCellsRequest true "Cells payload"
// @Success 204
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 422 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/{rateMatrixID}/cells [put]
func (h *Handler) replaceCells(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	matrixID, err := h.matrixID(c)
	if err != nil {
		return
	}

	body := new(replaceCellsRequest)
	if err = c.ShouldBindJSON(body); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.ReplaceCells(
		c.Request.Context(),
		&repositories.ReplaceRateMatrixCellsRequest{
			TenantInfo:   tenantOf(authCtx),
			RateMatrixID: matrixID,
			Cells:        body.Cells,
		},
		authCtx.UserID,
	); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary List density classification scales
// @ID listDensityScales
// @Tags Rate Matrices
// @Produce json
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.DensityScale]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/density-scales/ [get]
func (h *Handler) listDensityScales(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.DensityScale], error) {
			return h.service.ListDensityScales(
				c.Request.Context(),
				&repositories.ListRateMatrixRequest{Filter: req},
			)
		},
	)
}

// @Summary List rate matrix options
// @ID listRateMatrixOptions
// @Tags Rate Matrices
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]ratematrix.RateMatrix]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /rate-matrices/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*ratematrix.RateMatrix], error) {
			return h.service.SelectOptions(c.Request.Context(), req)
		},
	)
}

func (h *Handler) matrixID(c *gin.Context) (pulid.ID, error) {
	matrixID, err := pulid.MustParse(c.Param("rateMatrixID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return pulid.Nil, err
	}

	return matrixID, nil
}

func tenantOf(authCtx *authctx.AuthContext) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}
