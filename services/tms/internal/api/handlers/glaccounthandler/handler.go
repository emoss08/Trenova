package glaccounthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/glaccountservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *glaccountservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *glaccountservice.Service
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
	api := rg.Group("/gl-accounts")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceGeneralLedgerAccount.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:glAccountID",
		h.pm.RequirePermission(permission.ResourceGeneralLedgerAccount.String(), permission.OpRead),
		h.get,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(
			permission.ResourceGeneralLedgerAccount.String(),
			permission.OpCreate,
		),
		h.create,
	)
	api.PUT(
		"/:glAccountID/",
		h.pm.RequirePermission(
			permission.ResourceGeneralLedgerAccount.String(),
			permission.OpUpdate,
		),
		h.update,
	)
	api.PATCH(
		"/:glAccountID/",
		h.pm.RequirePermission(
			permission.ResourceGeneralLedgerAccount.String(),
			permission.OpUpdate,
		),
		h.patch,
	)
	api.DELETE(
		"/:glAccountID/",
		h.pm.RequirePermission(
			permission.ResourceGeneralLedgerAccount.String(),
			permission.OpDelete,
		),
		h.delete,
	)
	api.POST(
		"/bulk-update-status/",
		h.pm.RequirePermission(
			permission.ResourceGeneralLedgerAccount.String(),
			permission.OpUpdate,
		),
		h.bulkUpdateStatus,
	)

	selectOptions := api.Group("/select-options")
	selectOptions.GET("/", h.selectOptions)
	selectOptions.GET("/:glAccountID/", h.getOption)
}

// @Summary List GL accounts
// @ID listGLAccounts
// @Tags GL Accounts
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]glaccount.GLAccount]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*glaccount.GLAccount], error) {
			return h.service.List(
				c.Request.Context(),
				&repositories.ListGLAccountsRequest{
					Filter: req,
				},
			)
		},
	)
}

// @Summary Bulk update GL account statuses
// @ID bulkUpdateGLAccountStatus
// @Tags GL Accounts
// @Accept json
// @Produce json
// @Param request body repositories.BulkUpdateGLAccountStatusRequest true "Bulk status update request"
// @Success 200 {array} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/bulk-update-status/ [post]
func (h *Handler) bulkUpdateStatus(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	req := new(repositories.BulkUpdateGLAccountStatusRequest)
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

// @Summary Get a GL account option
// @ID getGLAccountOption
// @Tags GL Accounts
// @Produce json
// @Param glAccountID path string true "GL account ID"
// @Success 200 {object} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/select-options/{glAccountID}/ [get]
func (h *Handler) getOption(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	glAccountID, err := pulid.MustParse(c.Param("glAccountID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(c.Request.Context(), repositories.GetGLAccountByIDRequest{
		ID: glAccountID,
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

// @Summary List GL account options
// @ID listGLAccountOptions
// @Tags GL Accounts
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]glaccount.GLAccount]
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/select-options/ [get]
func (h *Handler) selectOptions(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewSelectQueryRequest(c, authCtx)

	pagination.SelectOptions(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*glaccount.GLAccount], error) {
			return h.service.SelectOptions(
				c.Request.Context(),
				&repositories.GLAccountSelectOptionsRequest{
					SelectQueryRequest: req,
				},
			)
		},
	)
}

// @Summary Get a GL account
// @ID getGLAccount
// @Tags GL Accounts
// @Produce json
// @Param glAccountID path string true "GL account ID"
// @Success 200 {object} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/{glAccountID} [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	glAccountID, err := pulid.MustParse(c.Param("glAccountID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetGLAccountByIDRequest{
			ID: glAccountID,
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

// @Summary Create a GL account
// @ID createGLAccount
// @Tags GL Accounts
// @Accept json
// @Produce json
// @Param request body glaccount.GLAccount true "GL account payload"
// @Success 201 {object} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/ [post]
func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(glaccount.GLAccount)
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

// @Summary Patch a GL account
// @ID patchGLAccount
// @Tags GL Accounts
// @Accept json
// @Produce json
// @Param glAccountID path string true "GL account ID"
// @Param request body glaccount.GLAccount true "GL account payload"
// @Success 200 {object} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/{glAccountID}/ [patch]
func (h *Handler) patch(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	glAccountID, err := pulid.MustParse(c.Param("glAccountID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	existing, err := h.service.Get(
		c.Request.Context(),
		repositories.GetGLAccountByIDRequest{
			ID: glAccountID,
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

// @Summary Update a GL account
// @ID updateGLAccount
// @Tags GL Accounts
// @Accept json
// @Produce json
// @Param glAccountID path string true "GL account ID"
// @Param request body glaccount.GLAccount true "GL account payload"
// @Success 200 {object} glaccount.GLAccount
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/{glAccountID}/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	glAccountID, err := pulid.MustParse(c.Param("glAccountID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(glaccount.GLAccount)
	entity.ID = glAccountID
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

// @Summary Delete a GL account
// @ID deleteGLAccount
// @Tags GL Accounts
// @Param glAccountID path string true "GL account ID"
// @Success 204 "No Content"
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /gl-accounts/{glAccountID}/ [delete]
func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	glAccountID, err := pulid.MustParse(c.Param("glAccountID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), repositories.DeleteGLAccountRequest{
		ID: glAccountID,
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
