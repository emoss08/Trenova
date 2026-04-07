package documentpacketrulehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/documentpacketruleservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documentpacketruleservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documentpacketruleservice.Service
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
	api := rg.Group("/document-packet-rules")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceDocumentType.String(), permission.OpRead),
		h.list,
	)
	api.POST(
		"/",
		h.pm.RequirePermission(permission.ResourceDocumentType.String(), permission.OpCreate),
		h.create,
	)
	api.PUT(
		"/:ruleID/",
		h.pm.RequirePermission(permission.ResourceDocumentType.String(), permission.OpUpdate),
		h.update,
	)
	api.DELETE(
		"/:ruleID/",
		h.pm.RequirePermission(permission.ResourceDocumentType.String(), permission.OpDelete),
		h.delete,
	)
}

func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error) {
			return h.service.List(c.Request.Context(), &repositories.ListDocumentPacketRulesRequest{
				Filter: req,
			})
		},
	)
}

func (h *Handler) create(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entity := new(documentpacketrule.DocumentPacketRule)
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

func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleID, err := pulid.MustParse(c.Param("ruleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity := new(documentpacketrule.DocumentPacketRule)
	authctx.AddContextToRequest(authCtx, entity)
	entity.ID = ruleID

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

func (h *Handler) delete(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	ruleID, err := pulid.MustParse(c.Param("ruleID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = h.service.Delete(c.Request.Context(), ruleID, pagination.TenantInfo{
		OrgID: authCtx.OrganizationID,
		BuID:  authCtx.BusinessUnitID,
	}, authCtx.UserID); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
