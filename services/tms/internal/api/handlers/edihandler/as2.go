package edihandler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/core/services/ediinboundservice"
	"github.com/gin-gonic/gin"
)

const as2MaxRequestBody = 32 << 20

func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.POST("/edi/as2/inbound/", h.receiveAS2Message)
}

func (h *Handler) receiveAS2Message(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, as2MaxRequestBody))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	request := &ediinboundservice.ReceiveAS2MessageRequest{
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
	result, err := h.inboundService.ReceiveAS2Message(c.Request.Context(), request)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	if result.IsMDN {
		c.Status(http.StatusNoContent)
		return
	}
	if result.AsyncMDN {
		h.sendAsyncAS2MDN(c.Request.Context(), request.ReceiptDeliveryOption, result)
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
	result *ediinboundservice.ReceiveAS2MessageResult,
) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(requestCtx), time.Minute)
	go func() {
		defer cancel()
		if err := h.inboundService.SendAsyncAS2MDN(ctx, returnURL, result); err != nil {
			h.inboundService.LogAsyncMDNFailure(returnURL, err)
		}
	}()
}
