package edihandler

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/edi/app/edidelivery"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	pagination.List(c, req, h.eh, func() (*pagination.ListResult[*edimessage.EDIMessage], error) {
		return h.messages.ListMessages(
			c.Request.Context(),
			&edimessageports.ListEDIMessagesRequest{
				Filter: req,
				TransactionSet: edidocument.TransactionSet(
					helpers.QueryString(c, "transactionSet", ""),
				),
				Direction: edidocument.DocumentDirection(
					helpers.QueryString(c, "direction", ""),
				),
				PartnerID:     partnerID,
				Status:        edimessage.MessageStatus(helpers.QueryString(c, "status", "")),
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
	message, err := h.messages.GetMessage(
		c.Request.Context(),
		edimessageports.GetEDIMessageByIDRequest{
			ID:         messageID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, message)
}

func (h *Handler) inspectMessage(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	inspection, err := h.messages.InspectMessage(
		c.Request.Context(),
		edimessageports.GetEDIMessageByIDRequest{
			ID:         messageID,
			TenantInfo: pagination.FromAuth(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, inspection)
}

func (h *Handler) retryMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.delivery.RetryMessageDelivery(
		c.Request.Context(),
		&edidelivery.RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, message)
}

func (h *Handler) replayMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	messageID, err := pulid.MustParse(c.Param("messageID"))
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	message, err := h.delivery.ReplayMessageDelivery(
		c.Request.Context(),
		&edidelivery.RetryMessageDeliveryRequest{
			MessageID:  messageID,
			TenantInfo: pagination.FromAuthAsUser(authCtx),
		},
	)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, message)
}

func (h *Handler) bulkRetryMessageDelivery(c *gin.Context) {
	authCtx := authctx.GetAuthContext(c)
	req := new(edidelivery.BulkRetryMessageDeliveryRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		h.eh.HandleError(c, err)
		return
	}
	req.TenantInfo = pagination.TenantInfo{
		OrgID:  authCtx.OrganizationID,
		BuID:   authCtx.BusinessUnitID,
		UserID: authCtx.UserID,
	}
	result, err := h.delivery.BulkRetryMessageDelivery(c.Request.Context(), req)
	if err != nil {
		h.eh.HandleError(c, err)
		return
	}
	helpers.JSON(c, http.StatusOK, result)
}
