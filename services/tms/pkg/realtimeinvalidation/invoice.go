package realtimeinvalidation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	InvoiceActionUpdated      = "updated"
	InvoiceActionPDFGenerated = "pdf.generated"
	InvoiceActionSendUpdated  = "send.updated"
	InvoiceActionEDIUpdated   = "edi.updated"
	InvoiceActionBalance      = "balance.updated"
)

type InvoiceSnapshot struct {
	ID                 pulid.ID              `json:"id"`
	Number             string                `json:"number"`
	CustomerID         pulid.ID              `json:"customerId"`
	BillingQueueItemID pulid.ID              `json:"billingQueueItemId"`
	Status             invoice.Status        `json:"status"`
	SendStatus         invoice.SendStatus    `json:"sendStatus"`
	LastSendError      string                `json:"lastSendError,omitempty"`
	PDFDocumentID      pulid.ID              `json:"pdfDocumentId,omitempty"`
	EDISendStatus      invoice.EDISendStatus `json:"ediSendStatus"`
	UpdatedAt          int64                 `json:"updatedAt"`
}

type InvoiceChange struct {
	Invoice *invoice.Invoice
	Actor   servicesport.AuditActor
	Action  string
}

func NewInvoiceSnapshot(entity *invoice.Invoice) *InvoiceSnapshot {
	if entity == nil {
		return nil
	}
	return &InvoiceSnapshot{
		ID:                 entity.ID,
		Number:             entity.Number,
		CustomerID:         entity.CustomerID,
		BillingQueueItemID: entity.BillingQueueItemID,
		Status:             entity.Status,
		SendStatus:         entity.SendStatus,
		LastSendError:      entity.LastSendError,
		PDFDocumentID:      entity.PDFDocumentID,
		EDISendStatus:      entity.EDISendStatus,
		UpdatedAt:          entity.UpdatedAt,
	}
}

func PublishInvoice(
	ctx context.Context,
	realtime servicesport.RealtimeService,
	change *InvoiceChange,
) error {
	if change == nil || change.Invoice == nil {
		return ErrPublishParamsRequired
	}
	if realtime == nil {
		return nil
	}
	action := change.Action
	if action == "" {
		action = InvoiceActionUpdated
	}
	return Publish(ctx, realtime, &PublishParams{
		OrganizationID: change.Invoice.OrganizationID,
		BusinessUnitID: change.Invoice.BusinessUnitID,
		ActorUserID:    change.Actor.UserID,
		ActorType:      change.Actor.PrincipalType,
		ActorID:        change.Actor.PrincipalID,
		ActorAPIKeyID:  change.Actor.APIKeyID,
		Resource:       permission.ResourceInvoice.String(),
		Action:         action,
		RecordID:       change.Invoice.ID,
		Entity:         NewInvoiceSnapshot(change.Invoice),
	})
}

func PublishInvoiceAfterCommit(
	ctx context.Context,
	realtime servicesport.RealtimeService,
	logger *zap.Logger,
	change *InvoiceChange,
) {
	if realtime == nil || change == nil || change.Invoice == nil {
		return
	}
	ports.AfterCommit(ctx, func(committedCtx context.Context) {
		if err := PublishInvoice(committedCtx, realtime, change); err != nil && logger != nil {
			logger.Warn(
				"failed to publish invoice invalidation",
				zap.String("invoiceId", change.Invoice.ID.String()),
				zap.String("action", change.Action),
				zap.Error(err),
			)
		}
	})
}
