package accountingcontrolhandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service              *accountingcontrolservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}

type Handler struct {
	service *accountingcontrolservice.Service
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
	api := rg.Group("/accounting-controls")
	api.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceAccountingControl.String(), permission.OpRead),
		h.get,
	)
	api.PUT(
		"/",
		h.pm.RequirePermission(permission.ResourceAccountingControl.String(), permission.OpUpdate),
		h.update,
	)
}

// @Summary Get accounting control settings
// @Description Returns the current accounting control document for the authenticated tenant.
// @ID getAccountingControl
// @Tags Accounting Controls
// @Produce json
// @Success 200 {object} accountingcontrol.AccountingControl
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accounting-controls/ [get]
func (h *Handler) get(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity, err := h.service.Get(c.Request.Context(), tenantports.GetAccountingControlRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, entity)
}

// @Summary Update accounting control settings
// @ID updateAccountingControl
// @Tags Accounting Controls
// @Accept json
// @Produce json
// @Param request body accountingcontrol.AccountingControl true "Accounting control payload"
// @Success 200 {object} accountingcontrol.AccountingControl
// @Failure 400 {object} helpers.ProblemDetail
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 403 {object} helpers.ProblemDetail
// @Failure 500 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /accounting-controls/ [put]
func (h *Handler) update(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	entity := new(accountingcontrol.AccountingControl)
	if err := authctx.BindJSON(c, authCtx, entity); err != nil {
		h.eh.HandleError(c, err)
		return
	}

	updatedEntity, err := h.service.Update(c.Request.Context(), entity, authCtx.UserID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	helpers.JSON(c, http.StatusOK, updatedEntity)
}
