package audithandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              services.AuditService
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service services.AuditService
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
	api := rg.Group("/audit-entries")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAuditLog.String(), permission.OpRead),
		h.list,
	)
	api.GET(
		"/:auditEntryID/",
		h.pm.RequirePermission(permission.ResourceAuditLog.String(), permission.OpRead),
		h.get,
	)

	api.GET("/resource/:resourceID/", h.listByResourceID)
}

// @Summary List audit entries
// @ID listAuditEntries
// @Tags Audit Entries
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]audit.Entry]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /audit-entries/ [get]
func (h *Handler) list(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*audit.Entry], error) {
		return h.service.List(
			c.Request.Context(),
			&repositories.ListAuditEntriesRequest{
				Filter: req,
			},
		)
	})
}

// @Summary Get an audit entry
// @ID getAuditEntry
// @Tags Audit Entries
// @Produce json
// @Param auditEntryID path string true "Audit entry ID"
// @Success 200 {object} audit.Entry
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /audit-entries/{auditEntryID}/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	entryID, err := pulid.MustParse(c.Param("auditEntryID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	entity, err := h.service.GetByID(
		c.Request.Context(),
		repositories.GetAuditEntryByIDOptions{
			EntryID: entryID,
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

	c.JSON(http.StatusOK, entity)
}

// @Summary List audit entries by resource ID
// @ID listAuditEntriesByResourceID
// @Tags Audit Entries
// @Accept json
// @Produce json
// @Param resourceID path string true "Resource ID"
// @Param query query string false "Search query"
// @Param limit query int false "Page size" minimum(1) maximum(100)
// @Param offset query int false "Page offset" minimum(0)
// @Success 200 {object} pagination.Response[[]audit.Entry]
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Router /audit-entries/resource/{resourceID}/ [get]
func (h *Handler) listByResourceID(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)

	resourceID, err := pulid.MustParse(c.Param("resourceID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	pagination.List(
		c,
		req,
		h.eh,
		func() (*pagination.ListResult[*audit.Entry], error) {
			return h.service.ListByResourceID(
				c.Request.Context(),
				&repositories.ListByResourceIDRequest{
					Filter:     req,
					ResourceID: resourceID,
				},
			)
		},
	)
}
