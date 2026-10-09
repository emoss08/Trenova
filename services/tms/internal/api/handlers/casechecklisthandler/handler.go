package casechecklisthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.CaseChecklistService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

// Handler serves how case checklists are laid out. They are part of how the
// organization bills, so reading them takes billing control read and
// changing them billing control update.
type Handler struct {
	service services.CaseChecklistService
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

//nolint:gocritic // fx passes params by value
func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/case-checklists")
	resource := permission.ResourceBillingControl.String()

	api.GET("/", h.pm.RequirePermission(resource, permission.OpRead), h.list)
	api.PUT("/", h.pm.RequirePermission(resource, permission.OpUpdate), h.save)
	api.DELETE("/:templateID/", h.pm.RequirePermission(resource, permission.OpUpdate), h.delete)
}

type listQuery struct {
	Kind deskcase.ChecklistKind `form:"kind"`
}

type saveBody struct {
	ID         pulid.ID               `json:"id"`
	Version    int64                  `json:"version"`
	Kind       deskcase.ChecklistKind `json:"kind"`
	CustomerID pulid.ID               `json:"customerId"`
	Items      deskcase.TemplateItems `json:"items"`
}

func (h *Handler) list(c *gin.Context) {
	var query listQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.List(c.Request.Context(), &services.ListCaseChecklistTemplatesRequest{
		TenantInfo: pagination.FromAuthAsUser(authctx.GetAuthContext(c)),
		Kind:       query.Kind,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) save(c *gin.Context) {
	var body saveBody
	if err := c.ShouldBindJSON(&body); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	authCtx := authctx.GetAuthContext(c)

	saved, err := h.service.Save(c.Request.Context(), &services.SaveCaseChecklistTemplateRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		UserID:     authCtx.UserID,
		ID:         body.ID,
		Version:    body.Version,
		Kind:       body.Kind,
		CustomerID: body.CustomerID,
		Items:      body.Items,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, saved)
}

func (h *Handler) delete(c *gin.Context) {
	templateID, err := pulid.Parse(c.Param("templateID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	authCtx := authctx.GetAuthContext(c)

	if err = h.service.Delete(c.Request.Context(), &services.DeleteCaseChecklistTemplateRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		UserID:     authCtx.UserID,
		ID:         templateID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
