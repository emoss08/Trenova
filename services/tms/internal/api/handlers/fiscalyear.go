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
	fiscalyearservice "github.com/emoss08/trenova/internal/core/services/fiscalyear"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type FiscalYearHandlerParams struct {
	fx.In

	Service      *fiscalyearservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type FiscalYearHandler struct {
	service      *fiscalyearservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewFiscalYearHandler(p FiscalYearHandlerParams) *FiscalYearHandler {
	return &FiscalYearHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *FiscalYearHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/fiscal-years/")

	// List and create
	api.GET("", h.pm.RequirePermission(permission.ResourceFiscalYear, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceFiscalYear, "create"), h.create)

	// Special gets (most specific first)
	api.GET("current/", h.pm.RequirePermission(permission.ResourceFiscalYear, "read"), h.getCurrent)
	api.GET(
		"year/:year/",
		h.pm.RequirePermission(permission.ResourceFiscalYear, "read"),
		h.getByYear,
	)

	// Standard CRUD by ID
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceFiscalYear, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceFiscalYear, "update"), h.update)
	api.DELETE(":id/", h.pm.RequirePermission(permission.ResourceFiscalYear, "delete"), h.delete)

	// Status operations
	api.PUT(":id/close/", h.pm.RequirePermission(permission.ResourceFiscalYear, "close"), h.close)
	api.PUT(":id/lock/", h.pm.RequirePermission(permission.ResourceFiscalYear, "lock"), h.lock)
	api.PUT(
		":id/unlock/",
		h.pm.RequirePermission(permission.ResourceFiscalYear, "unlock"),
		h.unlock,
	) // Added permission
	api.PUT(
		":id/activate/",
		h.pm.RequirePermission(permission.ResourceFiscalYear, "activate"),
		h.activate,
	)
}

func (h *FiscalYearHandler) list(c *gin.Context) {
	pagination.Handle[*accounting.FiscalYear](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*accounting.FiscalYear], error) {
			return h.service.List(c.Request.Context(), &repositories.ListFiscalYearRequest{
				Filter: opts,
				FilterOptions: repositories.FiscalYearFilterOptions{
					IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails"),
					Status:             helpers.QueryString(c, "status", ""),
					Year:               helpers.QueryInt(c, "year", 0),
					IsCurrent:          helpers.QueryBool(c, "isCurrent", false),
				},
			})
		})
}

func (h *FiscalYearHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetFiscalYearByIDRequest{
			FiscalYearID: id,
			OrgID:        authCtx.OrganizationID,
			BuID:         authCtx.BusinessUnitID,
			UserID:       authCtx.UserID,
			FilterOptions: repositories.FiscalYearFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) getByYear(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	year, err := strconv.Atoi(c.Param("year"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByYear(
		c.Request.Context(),
		&repositories.GetFiscalYearByYearRequest{
			Year:  year,
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
			FilterOptions: repositories.FiscalYearFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) getCurrent(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity, err := h.service.GetCurrent(
		c.Request.Context(),
		&repositories.GetCurrentFiscalYearRequest{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
			FilterOptions: repositories.FiscalYearFilterOptions{
				IncludeUserDetails: helpers.QueryBool(c, "includeUserDetails", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(accounting.FiscalYear)
	if err := c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	context.AddContextToRequest(authCtx, entity)
	entity, err := h.service.Create(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, entity)
}

func (h *FiscalYearHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(accounting.FiscalYear)
	if err = c.ShouldBindJSON(entity); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity.ID = id
	context.AddContextToRequest(authCtx, entity)

	entity, err = h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Delete(c.Request.Context(), &repositories.DeleteFiscalYearRequest{
		FiscalYearID: id,
		OrgID:        authCtx.OrganizationID,
		BuID:         authCtx.BusinessUnitID,
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *FiscalYearHandler) close(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Close(c.Request.Context(), &repositories.CloseFiscalYearRequest{
		FiscalYearID: id,
		OrgID:        authCtx.OrganizationID,
		BuID:         authCtx.BusinessUnitID,
		ClosedByID:   authCtx.UserID,
		ClosedAt:     utils.NowUnix(),
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) lock(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Lock(c.Request.Context(), &repositories.LockFiscalYearRequest{
		FiscalYearID: id,
		OrgID:        authCtx.OrganizationID,
		BuID:         authCtx.BusinessUnitID,
		LockedByID:   authCtx.UserID,
		LockedAt:     utils.NowUnix(),
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) unlock(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Unlock(c.Request.Context(), &repositories.UnlockFiscalYearRequest{
		FiscalYearID: id,
		OrgID:        authCtx.OrganizationID,
		BuID:         authCtx.BusinessUnitID,
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *FiscalYearHandler) activate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Activate(c.Request.Context(), &repositories.ActivateFiscalYearRequest{
		FiscalYearID: id,
		OrgID:        authCtx.OrganizationID,
		BuID:         authCtx.BusinessUnitID,
		UserID:       authCtx.UserID,
	})
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}
