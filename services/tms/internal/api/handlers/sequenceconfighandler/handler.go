package sequenceconfighandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/sequenceconfigservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *sequenceconfigservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *sequenceconfigservice.Service
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
	api := rg.Group("/sequence-configs")
	api.GET(
		"/",
		h.pm.RequirePermission(
			permission.ResourceSequenceConfig.String(),
			permission.OpRead,
		),
		h.get,
	)
	api.PUT(
		"/",
		h.pm.RequirePermission(
			permission.ResourceSequenceConfig.String(),
			permission.OpUpdate,
		),
		h.update,
	)
}

// @Summary Get sequence configuration settings
// @Description Returns the current sequence configuration document for the authenticated tenant.
// @ID getSequenceConfig
// @Tags Sequence Configs
// @Produce json
// @Success 200 {object} tenant.SequenceConfigDocument
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /sequence-configs/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	doc, err := h.service.Get(
		c.Request.Context(),
		repositories.GetSequenceConfigRequest{
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

	c.JSON(http.StatusOK, doc)
}

// @Summary Update sequence configuration settings
// @ID updateSequenceConfig
// @Tags Sequence Configs
// @Accept json
// @Produce json
// @Param request body tenant.SequenceConfigDocument true "Sequence configuration payload"
// @Success 200 {object} tenant.SequenceConfigDocument
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /sequence-configs/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	doc := new(tenant.SequenceConfigDocument)
	if err := c.ShouldBindJSON(doc); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	doc.OrganizationID = authCtx.OrganizationID
	doc.BusinessUnitID = authCtx.BusinessUnitID
	for _, cfg := range doc.Configs {
		if cfg == nil {
			continue
		}
		cfg.OrganizationID = authCtx.OrganizationID
		cfg.BusinessUnitID = authCtx.BusinessUnitID
	}

	updatedDoc, err := h.service.Update(c.Request.Context(), doc, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, updatedDoc)
}
