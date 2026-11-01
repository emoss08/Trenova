package invoicehandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Sending      invoiceports.InvoiceSendService
	ErrorHandler *helpers.ErrorHandler
	RateLimiter  *middleware.RateLimiter
}

type Handler struct {
	sending     invoiceports.InvoiceSendService
	eh          *helpers.ErrorHandler
	rateLimiter *middleware.RateLimiter
}

func New(p Params) *Handler {
	return &Handler{
		sending:     p.Sending,
		eh:          p.ErrorHandler,
		rateLimiter: p.RateLimiter,
	}
}

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/billing/invoices/shared-documents")
	api.Use(h.rateLimiter.ByPublicToken("token"))
	api.GET("/:token/download/", h.downloadSharedDocument)
}

func (h *Handler) downloadSharedDocument(c *gin.Context) {
	result, err := h.sending.DownloadSharedDocument(
		c.Request.Context(),
		&invoiceports.DownloadInvoiceDocumentRequest{Token: c.Param("token")},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", result.ContentDisposition)
	c.Data(http.StatusOK, result.ContentType, result.Body)
}
