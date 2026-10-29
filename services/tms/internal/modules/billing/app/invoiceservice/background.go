package invoiceservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"go.uber.org/zap"
)

const invoiceIDKey = "invoiceId"

const (
	autoPostFailedEventType = "invoice_auto_post_failed"
	invoiceRecordEntity     = "invoice"
)

// AttachGeneratedPDF records the document a background render produced as the
// invoice's PDF and tells anyone with the invoice open that it is ready.
func (s *Service) AttachGeneratedPDF(
	ctx context.Context,
	req *invoiceports.AttachGeneratedPDFRequest,
) (*invoice.Invoice, error) {
	current, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	previous := *current
	current.PDFDocumentID = req.DocumentID
	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	s.logAction(
		updated,
		req.Actor,
		permission.OpUpdate,
		&previous,
		updated,
		"Invoice PDF generated",
	)
	s.publishInvoiceChange(ctx, updated, req.Actor, invoicerealtime.RealtimeActionPDFGenerated)

	return updated, nil
}

// ReportAutoPostFailure tells the person who approved an invoice that it could
// not be posted for them and is still a draft, and refreshes anyone with it
// open. An invoice someone has since posted or voided by hand needs no report.
// The person notified already has a toast from the notification, so the event
// names them and their open invoice stays quiet. Reporting is best effort: the
// post has already failed, and a report that cannot be delivered must not hide
// that.
func (s *Service) ReportAutoPostFailure(
	ctx context.Context,
	req *invoiceports.ReportAutoPostFailureRequest,
) error {
	current, err := s.repo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to load the invoice whose auto-post failed",
			zap.String(invoiceIDKey, req.InvoiceID.String()),
			zap.Error(err),
		)
		return nil
	}
	if current.Status != invoice.StatusDraft {
		return nil
	}

	reason := strings.TrimSpace(req.Reason)
	snapshot := invoicerealtime.NewRealtimeSnapshot(current)
	snapshot.PostError = reason
	if s.notifyAutoPostFailed(ctx, req, current, reason) {
		snapshot.NotifiedUserID = req.Actor.UserID
	}

	realtimeinvalidation.PublishAfterCommit(ctx, s.realtime, s.l, realtimeinvalidation.ChangeOf(
		permission.ResourceInvoice.String(),
		current,
		invoicerealtime.RealtimeActionPostFailed,
		req.Actor,
		snapshot,
	))

	return nil
}

func (s *Service) notifyAutoPostFailed(
	ctx context.Context,
	req *invoiceports.ReportAutoPostFailureRequest,
	entity *invoice.Invoice,
	reason string,
) bool {
	if s.notificationService == nil || req.Actor.UserID.IsNil() {
		return false
	}

	targetUserID := req.Actor.UserID
	buID := entity.BusinessUnitID
	message := "It is still a draft. Fix what stopped it and post it from the invoice."
	if reason != "" {
		message = reason + " " + message
	}
	data := map[string]any{
		invoiceIDKey: entity.ID.String(),
		"number":     entity.Number,
		"reason":     reason,
	}
	if link, ok := productguide.RecordPath(invoiceRecordEntity, entity.ID.String()); ok {
		data["link"] = link
	}

	if _, err := s.notificationService.Create(ctx, &notification.Notification{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: &buID,
		TargetUserID:   &targetUserID,
		Channel:        notification.ChannelUser,
		EventType:      autoPostFailedEventType,
		Priority:       notification.PriorityHigh,
		Title:          "Invoice " + entity.Number + " was not posted",
		Message:        message,
		Data:           data,
		RelatedEntities: map[string]any{
			invoiceIDKey:         entity.ID.String(),
			"billingQueueItemId": entity.BillingQueueItemID.String(),
			"customerId":         entity.CustomerID.String(),
		},
		Source: "invoiceservice.ReportAutoPostFailure",
	}); err != nil {
		s.l.Error("failed to announce a failed invoice auto-post",
			zap.String(invoiceIDKey, entity.ID.String()),
			zap.Error(err),
		)
		return false
	}

	return true
}
