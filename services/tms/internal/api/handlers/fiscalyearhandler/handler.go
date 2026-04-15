package fiscalyearhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *fiscalyearservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *fiscalyearservice.Service
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
	api := rg.Group("/fiscal-years")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:fiscalYearID",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:fiscalYearID/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:fiscalYearID/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpUpdate),
		h.patch,
	)
	api.DELETE(
		"/:fiscalYearID/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpDelete),
		h.delete,
	)
	api.PUT(
		"/:fiscalYearID/close/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpClose),
		h.close,
	)
	api.GET(
		"/:fiscalYearID/close-blockers/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpRead),
		h.closeBlockers,
	)
	api.PUT(
		"/:fiscalYearID/activate/",
		h.pm.RequirePermission(permission.ResourceFiscalYear.String(), permission.OpActivate),
		h.activate,
	)
}

// @Summary List fiscal years
// @ID listFiscalYears
// @Tags Fiscal Years
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Param includePeriods query bool false "Include fiscal periods"
// @Success 200 {object} pagination.Response[[]fiscalyear.FiscalYear]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*fiscalyear.FiscalYear], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListFiscalYearsRequest{
					Filter:         req,
					IncludePeriods: helpers.QueryBool(c, "includePeriods", false),
				},
			)
		},
	)
}

// @Summary Get a fiscal year
// @ID getFiscalYear
// @Tags Fiscal Years
// @Produce json
// @Param fiscalYearID path string true "Fiscal year ID"
// @Success 200 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFiscalYearByIDRequest{
			ID: fiscalYearID,
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

func (h *Handler) closeBlockers(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.GetCloseBlockers(
		c.Request.Context(),
		repositories.GetFiscalYearByIDRequest{
			ID: fiscalYearID,
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

	c.JSON(http.StatusOK, result)
}

// @Summary Create a fiscal year
// @ID createFiscalYear
// @Tags Fiscal Years
// @Accept json
// @Produce json
// @Param request body fiscalyear.FiscalYear true "Fiscal year payload"
// @Success 201 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(fiscalyear.FiscalYear)
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

// @Summary Update a fiscal year
// @ID updateFiscalYear
// @Tags Fiscal Years
// @Accept json
// @Produce json
// @Param fiscalYearID path string true "Fiscal year ID"
// @Param request body fiscalyear.FiscalYear true "Fiscal year payload"
// @Success 200 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(fiscalyear.FiscalYear)
	entity.ID = fiscalYearID
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

// @Summary Patch a fiscal year
// @ID patchFiscalYear
// @Tags Fiscal Years
// @Accept json
// @Produce json
// @Param fiscalYearID path string true "Fiscal year ID"
// @Param request body fiscalyear.FiscalYear true "Fiscal year payload"
// @Success 200 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFiscalYearByIDRequest{
			ID: fiscalYearID,
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

	updatedEntity, err := h.service.Update(c.Request.Context(), existing, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedEntity)
}

// @Summary Delete a fiscal year
// @ID deleteFiscalYear
// @Tags Fiscal Years
// @Param fiscalYearID path string true "Fiscal year ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeleteFiscalYearRequest{
		ID: fiscalYearID,
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

// @Summary Close a fiscal year
// @ID closeFiscalYear
// @Tags Fiscal Years
// @Produce json
// @Param fiscalYearID path string true "Fiscal year ID"
// @Success 200 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID}/close/ [put]
func (h *Handler) close(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Close(c.Request.Context(), repositories.CloseFiscalYearRequest{
		ID: fiscalYearID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

// @Summary Activate a fiscal year
// @ID activateFiscalYear
// @Tags Fiscal Years
// @Produce json
// @Param fiscalYearID path string true "Fiscal year ID"
// @Success 200 {object} fiscalyear.FiscalYear
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-years/{fiscalYearID}/activate/ [put]
func (h *Handler) activate(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Activate(c.Request.Context(), repositories.ActivateFiscalYearRequest{
		ID: fiscalYearID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	}, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
