package fiscalperiodhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *fiscalperiodservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *fiscalperiodservice.Service
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
	api := rg.Group("/fiscal-periods")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:fiscalPeriodID",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:fiscalPeriodID/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpUpdate),
		h.update,
	)
	api.PATCH(
		"/:fiscalPeriodID/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpUpdate),
		h.patch,
	)
	api.DELETE(
		"/:fiscalPeriodID/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpDelete),
		h.delete,
	)
	api.PUT(
		"/:fiscalPeriodID/close/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpClose),
		h.close,
	)
	api.PUT(
		"/:fiscalPeriodID/reopen/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpReopen),
		h.reopen,
	)
	api.PUT(
		"/:fiscalPeriodID/lock/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpLock),
		h.lock,
	)
	api.PUT(
		"/:fiscalPeriodID/unlock/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod.String(), permission.OpUnlock),
		h.unlock,
	)
}

// @Summary List fiscal periods
// @ID listFiscalPeriods
// @Tags Fiscal Periods
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]fiscalperiod.FiscalPeriod]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*fiscalperiod.FiscalPeriod], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListFiscalPeriodsRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Get a fiscal period
// @ID getFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFiscalPeriodByIDRequest{
			ID: fiscalPeriodID,
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

// @Summary Create a fiscal period
// @ID createFiscalPeriod
// @Tags Fiscal Periods
// @Accept json
// @Produce json
// @Param request body fiscalperiod.FiscalPeriod true "Fiscal period payload"
// @Success 201 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(fiscalperiod.FiscalPeriod)
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

// @Summary Update a fiscal period
// @ID updateFiscalPeriod
// @Tags Fiscal Periods
// @Accept json
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Param request body fiscalperiod.FiscalPeriod true "Fiscal period payload"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(fiscalperiod.FiscalPeriod)
	entity.ID = fiscalPeriodID
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

// @Summary Patch a fiscal period
// @ID patchFiscalPeriod
// @Tags Fiscal Periods
// @Accept json
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Param request body fiscalperiod.FiscalPeriod true "Fiscal period payload"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetFiscalPeriodByIDRequest{
			ID: fiscalPeriodID,
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

// @Summary Delete a fiscal period
// @ID deleteFiscalPeriod
// @Tags Fiscal Periods
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeleteFiscalPeriodRequest{
		ID: fiscalPeriodID,
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

// @Summary Close a fiscal period
// @ID closeFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/close/ [put]
func (h *Handler) close(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Close(c.Request.Context(), repositories.CloseFiscalPeriodRequest{
		ID: fiscalPeriodID,
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

// @Summary Reopen a fiscal period
// @ID reopenFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/reopen/ [put]
func (h *Handler) reopen(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Reopen(c.Request.Context(), repositories.ReopenFiscalPeriodRequest{
		ID: fiscalPeriodID,
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

// @Summary Lock a fiscal period
// @ID lockFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/lock/ [put]
func (h *Handler) lock(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Lock(c.Request.Context(), repositories.LockFiscalPeriodRequest{
		ID: fiscalPeriodID,
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

// @Summary Unlock a fiscal period
// @ID unlockFiscalPeriod
// @Tags Fiscal Periods
// @Produce json
// @Param fiscalPeriodID path string true "Fiscal period ID"
// @Success 200 {object} fiscalperiod.FiscalPeriod
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /fiscal-periods/{fiscalPeriodID}/unlock/ [put]
func (h *Handler) unlock(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fiscalPeriodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Unlock(c.Request.Context(), repositories.UnlockFiscalPeriodRequest{
		ID: fiscalPeriodID,
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
