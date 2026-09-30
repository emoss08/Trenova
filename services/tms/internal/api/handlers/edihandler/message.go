package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

func (h *Handler) registerMessageRoutes(messages *gin.RouterGroup) {
	messages.GET(
		"/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.listMessages,
	)
	messages.GET(
		"/:messageID/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.getMessage,
	)
	messages.GET(
		"/:messageID/inspect/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpRead),
		h.inspectMessage,
	)
	messages.POST(
		"/:messageID/retry-delivery/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.retryMessageDelivery,
	)
	messages.POST(
		"/:messageID/replay/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.replayMessageDelivery,
	)
	messages.POST(
		"/bulk-retry-delivery/",
		h.pm.RequirePermission(permission.ResourceEDI.String(), permission.OpUpdate),
		h.bulkRetryMessageDelivery,
	)
}

func (h *Handler) listMessages(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := pagination.NewQueryOptions(c, authCtx)
	partnerID, _ := pulid.MustParse(helpers.QueryString(c, "partnerId", ""))
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edi.EDIMessage], error) {
		return h.service.ListMessages(
			c.Request.Context(),
			&repositories.ListEDIMessagesRequest{
				Filter:         req,
				TransactionSet: edi.TransactionSet(helpers.QueryString(c, "transactionSet", "")),
				Direction: edi.DocumentDirection(
					helpers.QueryString(c, "direction", ""),
				),
				PartnerID:     partnerID,
				Status:        edi.MessageStatus(helpers.QueryString(c, "status", "")),
				Query:         helpers.QueryStringTrimmed(c, "query", ""),
				GeneratedFrom: helpers.QueryInt64(c, "generatedFrom", 0),
				GeneratedTo:   helpers.QueryInt64(c, "generatedTo", 0),
			},
		)
	})
}

func (h *Handler) getMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.service.GetMessage(
		c.Request.Context(),
		repositories.GetEDIMessageByIDRequest{
			ID:         messageID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, message)
}

func (h *Handler) inspectMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	inspection, err := h.service.InspectMessage(
		c.Request.Context(),
		repositories.GetEDIMessageByIDRequest{
			ID:         messageID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, inspection)
}

func (h *Handler) retryMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.service.RetryMessageDelivery(
		c.Request.Context(),
		&ediservice.RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, message)
}

func (h *Handler) replayMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.service.ReplayMessageDelivery(
		c.Request.Context(),
		&ediservice.RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, message)
}

func (h *Handler) bulkRetryMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(ediservice.BulkRetryMessageDeliveryRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	result, err := h.service.BulkRetryMessageDelivery(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}
