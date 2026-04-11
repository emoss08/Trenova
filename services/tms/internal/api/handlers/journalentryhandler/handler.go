package journalentryhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/journalentryservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *journalentryservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *journalentryservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/journal-entries")
	api.GET("/:journalEntryID/", h.pm.RequirePermission(permission.ResourceJournalEntry.String(), permission.OpRead), h.getEntry)
	api.GET("/source/:sourceObjectType/:sourceObjectID/", h.pm.RequirePermission(permission.ResourceJournalEntry.String(), permission.OpRead), h.getSource)
}

func (h *Handler) getEntry(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	id, err := pulid.MustParse(c.Param("journalEntryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	entity, err := h.service.GetEntry(c.Request.Context(), pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}, id)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}

func (h *Handler) getSource(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	entity, err := h.service.GetSourceByObject(c.Request.Context(), pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}, c.Param("sourceObjectType"), c.Param("sourceObjectID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, entity)
}
