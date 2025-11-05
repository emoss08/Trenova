package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	glaccountservice "github.com/emoss08/trenova/internal/core/services/glaccount"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type GLAccountHandlerParams struct {
	fx.In

	Service      *glaccountservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type GLAccountHandler struct {
	service      *glaccountservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewGLAccountHandler(p GLAccountHandlerParams) *GLAccountHandler {
	return &GLAccountHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *GLAccountHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/gl-accounts/")

	// List and create
	api.GET("", h.pm.RequirePermission(permission.ResourceGlAccount, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceGlAccount, "create"), h.create)

	// Special gets (most specific first)
	api.GET(
		"hierarchy/",
		h.pm.RequirePermission(permission.ResourceGlAccount, "read"),
		h.getHierarchy,
	)
	api.GET(
		"code/:code/",
		h.pm.RequirePermission(permission.ResourceGlAccount, "read"),
		h.getByCode,
	)
	api.GET(
		"type/:accountTypeId/",
		h.pm.RequirePermission(permission.ResourceGlAccount, "read"),
		h.getByType,
	)
	api.GET(
		"parent/:parentId/",
		h.pm.RequirePermission(permission.ResourceGlAccount, "read"),
		h.getByParent,
	)

	// Bulk operations
	api.POST(
		"bulk/",
		h.pm.RequirePermission(permission.ResourceGlAccount, "import"),
		h.bulkCreate,
	)

	// Standard CRUD by ID
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceGlAccount, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceGlAccount, "update"), h.update)
	api.DELETE(":id/", h.pm.RequirePermission(permission.ResourceGlAccount, "delete"), h.delete)
}

func (h *GLAccountHandler) list(c *gin.Context) {
	var isActive, isSystem, allowManualJE *bool

	if c.Query("isActive") != "" {
		val := helpers.QueryBool(c, "isActive")
		isActive = &val
	}
	if c.Query("isSystem") != "" {
		val := helpers.QueryBool(c, "isSystem")
		isSystem = &val
	}
	if c.Query("allowManualJE") != "" {
		val := helpers.QueryBool(c, "allowManualJE")
		allowManualJE = &val
	}

	pagination.Handle[*accounting.GLAccount](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*accounting.GLAccount], error) {
			return h.service.List(c.Request.Context(), &repositories.ListGLAccountRequest{
				Filter: opts,
				FilterOptions: &repositories.GLAccountFilterOptions{
					IncludeAccountType: helpers.QueryBool(c, "includeAccountType"),
					IncludeParent:      helpers.QueryBool(c, "includeParent"),
					IncludeChildren:    helpers.QueryBool(c, "includeChildren"),
					Status:             helpers.QueryString(c, "status", ""),
					AccountTypeID:      helpers.QueryString(c, "accountTypeId", ""),
					ParentID:           helpers.QueryString(c, "parentId", ""),
					IsActive:           isActive,
					IsSystem:           isSystem,
					AllowManualJE:      allowManualJE,
				},
			})
		})
}

func (h *GLAccountHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		&repositories.GetGLAccountByIDRequest{
			GLAccountID: id,
			OrgID:       authCtx.OrganizationID,
			BuID:        authCtx.BusinessUnitID,
			UserID:      authCtx.UserID,
			FilterOptions: &repositories.GLAccountFilterOptions{
				IncludeAccountType: helpers.QueryBool(c, "includeAccountType", false),
				IncludeParent:      helpers.QueryBool(c, "includeParent", false),
				IncludeChildren:    helpers.QueryBool(c, "includeChildren", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *GLAccountHandler) getByCode(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Account code is required"})
		return
	}

	entity, err := h.service.GetByCode(
		c.Request.Context(),
		&repositories.GetGLAccountByCodeRequest{
			AccountCode: code,
			OrgID:       authCtx.OrganizationID,
			BuID:        authCtx.BusinessUnitID,
			FilterOptions: &repositories.GLAccountFilterOptions{
				IncludeAccountType: helpers.QueryBool(c, "includeAccountType", false),
				IncludeParent:      helpers.QueryBool(c, "includeParent", false),
				IncludeChildren:    helpers.QueryBool(c, "includeChildren", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *GLAccountHandler) getByType(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	accountTypeID, err := pulid.MustParse(c.Param("accountTypeId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entities, err := h.service.GetByType(
		c.Request.Context(),
		&repositories.GetGLAccountsByTypeRequest{
			AccountTypeID: accountTypeID,
			OrgID:         authCtx.OrganizationID,
			BuID:          authCtx.BusinessUnitID,
			FilterOptions: &repositories.GLAccountFilterOptions{
				IncludeAccountType: helpers.QueryBool(c, "includeAccountType", false),
				IncludeParent:      helpers.QueryBool(c, "includeParent", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *GLAccountHandler) getByParent(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	parentID, err := pulid.MustParse(c.Param("parentId"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entities, err := h.service.GetByParent(
		c.Request.Context(),
		&repositories.GetGLAccountsByParentRequest{
			ParentID: parentID,
			OrgID:    authCtx.OrganizationID,
			BuID:     authCtx.BusinessUnitID,
			FilterOptions: &repositories.GLAccountFilterOptions{
				IncludeAccountType: helpers.QueryBool(c, "includeAccountType", false),
				IncludeChildren:    helpers.QueryBool(c, "includeChildren", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *GLAccountHandler) getHierarchy(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entities, err := h.service.GetHierarchy(
		c.Request.Context(),
		&repositories.GetGLAccountHierarchyRequest{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
			FilterOptions: &repositories.GLAccountFilterOptions{
				IncludeAccountType: helpers.QueryBool(c, "includeAccountType", false),
			},
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entities)
}

func (h *GLAccountHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var entity accounting.GLAccount
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

func (h *GLAccountHandler) bulkCreate(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	var entities []*accounting.GLAccount
	if err := c.ShouldBindJSON(&entities); err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	for _, entity := range entities {
		entity.OrganizationID = authCtx.OrganizationID
		entity.BusinessUnitID = authCtx.BusinessUnitID
	}

	err := h.service.BulkCreate(
		c.Request.Context(),
		entities,
		authCtx.UserID,
		authCtx.OrganizationID,
		authCtx.BusinessUnitID,
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "GL accounts created successfully",
		"count":   len(entities),
	})
}

func (h *GLAccountHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	var entity accounting.GLAccount
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

func (h *GLAccountHandler) delete(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	err = h.service.Delete(
		c.Request.Context(),
		&repositories.DeleteGLAccountRequest{
			GLAccountID: id,
			OrgID:       authCtx.OrganizationID,
			BuID:        authCtx.BusinessUnitID,
			UserID:      authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "GL account deleted successfully"})
}
