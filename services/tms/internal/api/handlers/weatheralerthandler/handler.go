package weatheralerthandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Service      serviceports.WeatherAlertService
	ErrorHandler *helpers.ErrorHandler
}

type Handler struct {
	service serviceports.WeatherAlertService
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.Service,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/weather-alerts")
	api.GET("/", h.list)
	api.GET("/:alertID/", h.get)
}

func (h *Handler) list(c *gin.Context) {
	ac := authctx.GetAuthContext(c)

	result, err := h.service.GetActiveAlerts(c.Request.Context(), pagination.TenantInfo{
		OrgID:  ac.OrganizationID,
		BuID:   ac.BusinessUnitID,
		UserID: ac.UserID,
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) get(c *gin.Context) {
	ac := authctx.GetAuthContext(c)
	alertID, err := pulid.MustParse(c.Param("alertID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	result, err := h.service.GetAlertDetail(c.Request.Context(), &serviceports.GetWeatherAlertDetailRequest{
		ID: alertID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  ac.OrganizationID,
			BuID:   ac.BusinessUnitID,
			UserID: ac.UserID,
		},
	})
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}
