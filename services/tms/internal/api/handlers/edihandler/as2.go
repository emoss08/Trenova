package edihandler

import (
	"context"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundas2"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/gin-gonic/gin"
)

const (
	as2MaxRequestBody = 32 << 20
	as2InboundRoute   = edi.AS2InboundRoot + ":as2InboundToken/"
)

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST(as2InboundRoute, h.receiveAS2Message)
}

func (h *Handler) receiveAS2Message(c *gin.Context) {
	body, err := helpers.ReadBoundedRequestBody(c, as2MaxRequestBody)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	request := &ediinboundas2.ReceiveAS2MessageRequest{
		InboundToken:                   c.Param("as2InboundToken"),
		From:                           c.GetHeader("AS2-From"),
		To:                             c.GetHeader("AS2-To"),
		MessageID:                      c.GetHeader("Message-ID"),
		ContentType:                    c.GetHeader("Content-Type"),
		TransferEncoding:               c.GetHeader("Content-Transfer-Encoding"),
		DispositionNotificationTo:      c.GetHeader("Disposition-Notification-To"),
		DispositionNotificationOptions: c.GetHeader("Disposition-Notification-Options"),
		ReceiptDeliveryOption:          c.GetHeader("Receipt-Delivery-Option"),
		Body:                           body,
	}
	result, err := h.as2Service.ReceiveAS2Message(c.Request.Context(), request)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if result.IsMDN {
		c.Status(http.StatusNoContent)
		return
	}
	if result.AsyncMDNURL != "" {
		h.sendAsyncAS2MDN(c.Request.Context(), result.AsyncMDNURL, result)
		c.Status(http.StatusOK)
		return
	}
	for key, values := range result.MDNHeaders {
		for _, value := range values {
			c.Header(key, value)
		}
	}
	c.Data(http.StatusOK, result.MDNContentType, result.MDNBody)
}

func (h *Handler) sendAsyncAS2MDN(
	requestCtx context.Context,
	returnURL string,
	result *ediinboundas2.ReceiveAS2MessageResult,
) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), time.Minute)
	go func() {
		defer cancel()
		if err := h.as2Service.SendAsyncAS2MDN(ctx, returnURL, result); err != nil {
			h.as2Service.LogAsyncMDNFailure(returnURL, err)
		}
	}()
}
