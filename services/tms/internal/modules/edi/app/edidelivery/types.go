package edidelivery

import (
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type DeliverEDIMessageWorkflowPayload struct {
	MessageID   pulid.ID                          `json:"messageId"`
	TenantInfo  pagination.TenantInfo             `json:"tenantInfo"`
	RetryPolicy *editransport.DeliveryRetryPolicy `json:"retryPolicy,omitempty"`
}

type DeliverEDIMessageWorkflowResult struct {
	MessageID      pulid.ID                         `json:"messageId"`
	DeliveryStatus edimessage.MessageDeliveryStatus `json:"deliveryStatus"`
	RemotePath     string                           `json:"remotePath"`
}

type MarkEDIMessageDeadLetteredPayload struct {
	MessageID  pulid.ID              `json:"messageId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Reason     string                `json:"reason"`
}

type RetryMessageDeliveryRequest struct {
	MessageID  pulid.ID              `json:"-"`
	TenantInfo pagination.TenantInfo `json:"-"`
}
