package handlers

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	documenttypeservice "github.com/emoss08/trenova/internal/core/services/documenttype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type DocumentTypeHandlerParams struct {
	fx.In

	Service      *documenttypeservice.Service
	PM           *middleware.PermissionMiddleware
	ErrorHandler *helpers.ErrorHandler
}

type DocumentTypeHandler struct {
	service      *documenttypeservice.Service
	pm           *middleware.PermissionMiddleware
	errorHandler *helpers.ErrorHandler
}

func NewDocumentTypeHandler(p DocumentTypeHandlerParams) *DocumentTypeHandler {
	return &DocumentTypeHandler{
		service:      p.Service,
		pm:           p.PM,
		errorHandler: p.ErrorHandler,
	}
}

func (h *DocumentTypeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/document-types/")
	api.GET("", h.pm.RequirePermission(permission.ResourceDocumentType, "read"), h.list)
	api.POST("", h.pm.RequirePermission(permission.ResourceDocumentType, "create"), h.create)
	api.GET(":id/", h.pm.RequirePermission(permission.ResourceDocumentType, "read"), h.get)
	api.PUT(":id/", h.pm.RequirePermission(permission.ResourceDocumentType, "update"), h.update)
}

func (h *DocumentTypeHandler) list(c *gin.Context) {
	pagination.Handle[*documenttype.DocumentType](c, context.GetAuthContext(c)).
		WithErrorHandler(h.errorHandler).
		Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*documenttype.DocumentType], error) {
			return h.service.List(c.Request.Context(), &repositories.ListDocumentTypeRequest{
				Filter: opts,
			})
		})
}

func (h *DocumentTypeHandler) get(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity, err := h.service.Get(
		c.Request.Context(),
		repositories.GetDocumentTypeByIDRequest{
			ID:     id,
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
	)
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, entity)
}

func (h *DocumentTypeHandler) create(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	entity := new(documenttype.DocumentType)
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

func (h *DocumentTypeHandler) update(c *gin.Context) {
	authCtx := context.GetAuthContext(c)

	id, err := pulid.MustParse(c.Param("id"))
	if err != nil {
		h.errorHandler.HandleError(c, err)
		return
	}

	entity := new(documenttype.DocumentType)
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
