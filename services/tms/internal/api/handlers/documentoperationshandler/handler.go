package documentoperationshandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/services/documentoperationsservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *documentoperationsservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *documentoperationsservice.Service
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
	api := rg.Group("/admin/document-operations")
	api.Use(h.pm.RequirePlatformAdmin())
	api.GET("/:documentID/", h.getDiagnostics)
	api.POST("/:documentID/reextract/", h.reextract)
	api.POST("/:documentID/regenerate-preview/", h.regeneratePreview)
	api.POST("/:documentID/resync-search/", h.resyncSearch)
}

func (h *Handler) getDiagnostics(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.GetDiagnostics(
		c.Request.Context(),
		documentID,
		pagination.TenantInfo{
			OrgID: authCtx.OrganizationID,
			BuID:  authCtx.BusinessUnitID,
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) reextract(c *gin.Context) {
	h.runAction(c, func(documentID pulid.ID, tenantInfo pagination.TenantInfo) error {
		return h.service.Reextract(c.Request.Context(), documentID, tenantInfo)
	})
}

func (h *Handler) regeneratePreview(c *gin.Context) {
	h.runAction(c, func(documentID pulid.ID, tenantInfo pagination.TenantInfo) error {
		return h.service.RegeneratePreview(c.Request.Context(), documentID, tenantInfo)
	})
}

func (h *Handler) resyncSearch(c *gin.Context) {
	h.runAction(c, func(documentID pulid.ID, tenantInfo pagination.TenantInfo) error {
		return h.service.ResyncSearch(c.Request.Context(), documentID, tenantInfo)
	})
}

func (h *Handler) runAction(
	c *gin.Context,
	fn func(documentID pulid.ID, tenantInfo pagination.TenantInfo) error,
) {
	authCtx := authctx.GetAuthContext(c)
	documentID, err := pulid.MustParse(c.Param("documentID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	if err = fn(documentID, pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}
