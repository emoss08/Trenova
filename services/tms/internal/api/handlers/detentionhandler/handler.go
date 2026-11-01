package detentionhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionsimulation"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Review               *detentionreview.Service
	Simulation           *detentionsimulation.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	review     *detentionreview.Service
	simulation *detentionsimulation.Service
	eh         *helpers.ErrorHandler
	pm         *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{
		review:     p.Review,
		simulation: p.Simulation,
		eh:         p.ErrorHandler,
		pm:         p.PermissionMiddleware,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	resource := permission.ResourceDetention.String()

	api := rg.Group("/detention")
	api.GET("/desk/", h.pm.RequirePermission(resource, permission.OpRead), h.desk)
}

func (h *Handler) tenant(c *gin.Context) pagination.TenantInfo {
	authCtx := authctx.GetAuthContext(c)
	return pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
}

// @Summary List the live detention desk
// @ID listDetentionDesk
// @Tags Detention
// @Produce json
// @Success 200 {array} detentionreview.DeskEntry
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /detention/desk/ [get]
func (h *Handler) desk(c *gin.Context) {
	entries, err := h.review.ListDesk(c.Request.Context(), h.tenant(c))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entries)
}
