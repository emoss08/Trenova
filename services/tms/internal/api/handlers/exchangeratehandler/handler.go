package exchangeratehandler

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/exchangerateservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

const defaultDateLayout = "2006-01-02"

type Params struct {
	fx.In

	ExchangeRateService *exchangerateservice.Service
	ErrorHandler        *helpers.ErrorHandler
}

type Handler struct {
	service *exchangerateservice.Service
	eh      *helpers.ErrorHandler
}

func New(p Params) *Handler {
	return &Handler{
		service: p.ExchangeRateService,
		eh:      p.ErrorHandler,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/exchange-rates")
	api.GET("/convert", h.convert)
	api.GET("/latest", h.latest)
	api.POST("/refresh", h.refresh)
}

func (h *Handler) convert(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)

	fromCurrency := c.Query("from")
	toCurrency := c.Query("to")
	amountStr := c.Query("amount")
	dateStr := c.DefaultQuery("date", time.Now().Format(defaultDateLayout))

	if fromCurrency == "" || toCurrency == "" || amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from, to, and amount query parameters are required"})
		return
	}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid amount"})
		return
	}

	date, err := time.Parse(defaultDateLayout, dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DD"})
		return
	}

	result, err := h.service.Convert(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		fromCurrency,
		toCurrency,
		amount,
		date,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) latest(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	baseCurrency := c.DefaultQuery("base", "USD")

	result, err := h.service.GetLatestRates(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		baseCurrency,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) refresh(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	baseCurrency := c.DefaultQuery("base", "USD")

	err := h.service.RefreshRates(
		c.Request.Context(),
		pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		},
		baseCurrency,
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
