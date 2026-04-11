package glbalancehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/glbalanceservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In
	Service              *glbalanceservice.Service
	ErrorHandler         *helpers.ErrorHandler
	PermissionMiddleware *middleware.PermissionMiddleware
}
type Handler struct {
	service *glbalanceservice.Service
	eh      *helpers.ErrorHandler
	pm      *middleware.PermissionMiddleware
}

func New(p Params) *Handler {
	return &Handler{service: p.Service, eh: p.ErrorHandler, pm: p.PermissionMiddleware}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/accounting/trial-balance")
	api.GET("/:fiscalPeriodID/", h.pm.RequirePermission(permission.ResourceGeneralLedgerAccount.String(), permission.OpRead), h.listByPeriod)
}

func (h *Handler) listByPeriod(c *gin.Context) {
	auth := authctx.GetAuthContext(c)
	periodID, err := pulid.MustParse(c.Param("fiscalPeriodID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	balances, err := h.service.ListTrialBalanceByPeriod(c.Request.Context(), pagination.TenantInfo{OrgID: auth.OrganizationID, BuID: auth.BusinessUnitID, UserID: auth.UserID}, periodID)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, balances)
}
