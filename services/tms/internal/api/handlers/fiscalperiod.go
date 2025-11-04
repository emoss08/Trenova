package handlers

import (
	"net/http"
	"strconv"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	fiscalperiodservice "github.com/emoss08/trenova/internal/core/services/fiscalperiod"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type FiscalPeriodHandlerParams struct {
	fx.In

	Service      *fiscalperiodservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type FiscalPeriodHandler struct {
	service      *fiscalperiodservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewFiscalPeriodHandler(p FiscalPeriodHandlerParams) *FiscalPeriodHandler {
	return &FiscalPeriodHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *FiscalPeriodHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/fiscal-periods/")

	// List and create
	api.GET("", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "create"), h.create)

	// Special gets (most specific first)
	api.GET(
		"fiscal-year/:fiscalYearId/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"),
		h.getByFiscalYear,
	)
	api.GET(
		"fiscal-year/:fiscalYearId/period/:periodNumber/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"),
		h.getByNumber,
	)

	// Standard CRUD by ID
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "update"), h.update)
	api.DELETE(
		":id/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod, "delete"),
		h.delete,
	)

	// Status operations
	api.PUT(":id/close/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "close"), h.close)
	api.PUT(
		":id/reopen/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod, "close"),
		h.reopen,
	)
	api.PUT(":id/lock/", h.pm.RequirePermission(permission.ResourceFiscalPeriod, "lock"), h.lock)
	api.PUT(
		":id/unlock/",
		h.pm.RequirePermission(permission.ResourceFiscalPeriod, "unlock"),
		h.unlock,
	)
}

func (h *FiscalPeriodHandler) list(c *gin.Context) {
	pagination.Handle[*accounting.FiscalPeriod](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*accounting.FiscalPeriod], error) {
			return h.service.List(c.Request.Context(), &repositories.ListFiscalPeriodRequest{
				Filter: opts,
				FilterOptions: repositories.FiscalPeriodFilterOptions{
					IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails"),
					IncludeFiscalYear:  helpers.QueryBool(c, "includeFiscalYear"),
					Status:             helpers.QueryString(c, "status", ""),
					FiscalYearID:       helpers.QueryString(c, "fiscalYearId", ""),
					PeriodNumber:       helpers.QueryInt(c, "periodNumber", 0),
				},
			})
		})
}

func (h *FiscalPeriodHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetFiscalPeriodByIDRequest{
			FiscalPeriodID: id,
			OrgID:          authCtx.OrganizationID,
			BuID:           authCtx.BusinessUnitID,
			UserID:         authCtx.UserID,
			FilterOptions: repositories.FiscalPeriodFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
				IncludeFiscalYear:  helpers.QueryBool(c, "includeFiscalYear", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalPeriodHandler) getByNumber(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	periodNumber, err := strconv.Atoi(c.Param("periodNumber"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByNumber(
		c.Request.Context(),
		&repositories.GetFiscalPeriodByNumberRequest{
			FiscalYearID: fiscalYearID,
			PeriodNumber: periodNumber,
			OrgID:        authCtx.OrganizationID,
			BuID:         authCtx.BusinessUnitID,
			FilterOptions: repositories.FiscalPeriodFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
				IncludeFiscalYear:  helpers.QueryBool(c, "includeFiscalYear", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalPeriodHandler) getByFiscalYear(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	fiscalYearID, err := pulid.MustParse(c.Param("fiscalYearId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entities, err := h.service.GetByFiscalYear(
		c.Request.Context(),
		&repositories.GetFiscalPeriodsByYearRequest{
			FiscalYearID: fiscalYearID,
			OrgID:        authCtx.OrganizationID,
			BuID:         authCtx.BusinessUnitID,
			FilterOptions: repositories.FiscalPeriodFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
				IncludeFiscalYear:  helpers.QueryBool(c, "includeFiscalYear", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *FiscalPeriodHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var entity accounting.FiscalPeriod
	if err := c.ShouldBindJSON(&entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	created, err := h.service.Create(c.Request.Context(), &entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *FiscalPeriodHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	var entity accounting.FiscalPeriod
	if err = c.ShouldBindJSON(&entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	entity.OrganizationID = authCtx.OrganizationID
	entity.BusinessUnitID = authCtx.BusinessUnitID

	updated, err := h.service.Update(c.Request.Context(), &entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *FiscalPeriodHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), &repositories.DeleteFiscalPeriodRequest{
		FiscalPeriodID: id,
		OrgID:          authCtx.OrganizationID,
		BuID:           authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *FiscalPeriodHandler) close(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	closed, err := h.service.Close(c.Request.Context(), &repositories.CloseFiscalPeriodRequest{
		FiscalPeriodID: id,
		OrgID:          authCtx.OrganizationID,
		BuID:           authCtx.BusinessUnitID,
		ClosedByID:     authCtx.UserID,
		ClosedAt:       utils.NowUnix(),
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, closed)
}

func (h *FiscalPeriodHandler) reopen(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	reopened, err := h.service.Reopen(c.Request.Context(), &repositories.ReopenFiscalPeriodRequest{
		FiscalPeriodID: id,
		OrgID:          authCtx.OrganizationID,
		BuID:           authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, reopened)
}

func (h *FiscalPeriodHandler) lock(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	locked, err := h.service.Lock(c.Request.Context(), &repositories.LockFiscalPeriodRequest{
		FiscalPeriodID: id,
		OrgID:          authCtx.OrganizationID,
		BuID:           authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, locked)
}

func (h *FiscalPeriodHandler) unlock(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	unlocked, err := h.service.Unlock(c.Request.Context(), &repositories.UnlockFiscalPeriodRequest{
		FiscalPeriodID: id,
		OrgID:          authCtx.OrganizationID,
		BuID:           authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, unlocked)
}
