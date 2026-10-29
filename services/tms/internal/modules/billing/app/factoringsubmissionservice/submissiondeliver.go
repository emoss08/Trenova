package factoringsubmissionservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringtransportservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/factoringtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const scheduleSendKeyInfix = "-send-"

func (s *SubmissionService) Deliver(
	ctx context.Context,
	req *factoringsubmissionports.DeliverSubmissionRequest,
) (*factoringsubmissionports.DeliverSubmissionResult, error) {
	sub, err := s.repo.GetByID(ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:           req.SubmissionID,
		TenantInfo:   req.TenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if sub.Status != factoringsubmission.SubmissionStatusSending ||
		sub.SendAttemptGroup != req.Group {
		return &factoringsubmissionports.DeliverSubmissionResult{
			Status:  sub.Status,
			Skipped: true,
		}, nil
	}

	transport, err := s.transports.Get(ctx, factoringsubmissionports.GetTransportRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: sub.FactoringCompanyID,
	})
	if err != nil {
		return nil, err
	}
	sender, ok := s.senders[transport.Method]
	if !ok {
		return nil, errortypes.NewBusinessError(
			"{0} delivery is not available",
			transport.Method.String(),
		)
	}
	secret, err := factoringtransportservice.OpenSecret(ctx, s.cipher, req.TenantInfo, transport)
	if err != nil {
		return nil, err
	}
	files, err := s.deliveryFiles(ctx, req.TenantInfo, sub, transport)
	if err != nil {
		return nil, err
	}
	render, err := s.messageRenderer(ctx, req.TenantInfo, sub)
	if err != nil {
		return nil, err
	}

	result, err := sender.Send(ctx, &factoringsubmissionports.TransportSendRequest{
		TenantInfo:     req.TenantInfo,
		Submission:     sub,
		Transport:      transport,
		Secret:         secret,
		Files:          files,
		Render:         render,
		IdempotencyKey: SendKey(sub),
	})
	if err != nil {
		return nil, err
	}

	previous := factoringaction.CloneSubmission(sub)
	sub.TransportMethod = transport.Method
	sub.TransportTarget = transport.Target()
	if err = sub.MarkSent(s.now(), result.Reference, req.Attempt); err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, sub)
	if err != nil {
		return nil, err
	}

	actor := factoringaction.SystemActor(req.TenantInfo)
	s.rec.Log(&actor, &factoringaction.AuditEntry{
		Resource: submissionResource,
		Entity:   updated,
		Previous: previous,
		Op:       permission.OpUpdate,
		Comment:  "Schedule " + updated.Number + " sent to " + updated.TransportTarget,
	})
	s.rec.Publish(ctx, submissionResource, updated, nil, submissionRealtimeEvent)
	s.publishEvent(ctx, req.TenantInfo, updated, agenttrigger.EventFactoringSubmissionSent,
		"Schedule "+updated.Number+" of "+strconv.Itoa(updated.InvoiceCount)+
			" invoices was sent to "+updated.TransportTarget)

	return &factoringsubmissionports.DeliverSubmissionResult{
		Status:    updated.Status,
		Reference: result.Reference,
		Parts:     result.Parts,
	}, nil
}

func (s *SubmissionService) MarkFailed(
	ctx context.Context,
	req *factoringsubmissionports.FailSubmissionRequest,
) error {
	sub, err := s.repo.GetByID(ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:         req.SubmissionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return err
	}
	if sub.Status != factoringsubmission.SubmissionStatusSending ||
		sub.SendAttemptGroup != req.Group {
		return nil
	}
	previous := *sub
	if err = sub.MarkFailed(s.now(), req.Cause, req.Attempts); err != nil {
		return err
	}
	updated, err := s.repo.Update(ctx, sub)
	if err != nil {
		return err
	}

	actor := factoringaction.SystemActor(req.TenantInfo)
	s.rec.Log(&actor, &factoringaction.AuditEntry{
		Resource: submissionResource,
		Entity:   updated,
		Previous: &previous,
		Op:       permission.OpUpdate,
		Comment:  "Schedule " + updated.Number + " could not be sent: " + updated.LastError,
	})
	s.rec.Publish(ctx, submissionResource, updated, nil, submissionRealtimeEvent)
	s.publishEvent(ctx, req.TenantInfo, updated, agenttrigger.EventFactoringSubmissionFailed,
		"Schedule "+updated.Number+" could not be sent to "+updated.TransportTarget+
			": "+updated.LastError)
	return nil
}

func (s *SubmissionService) deliveryFiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
	transport *factoringsubmission.Transport,
) ([]factoringsubmissionports.TransportFile, error) {
	if sub.SchedulePDFDocumentID.IsNil() || sub.ScheduleCSVDocumentID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"Schedule {0} has not been built yet",
			sub.Number,
		)
	}
	attach := transport.Method == factoringsubmission.TransportMethodSftp || transport.AttachPackets
	files := make([]factoringsubmissionports.TransportFile, 0, len(sub.Items)+2)
	read := func(kind factoringsubmissionports.TransportFileKind, id pulid.ID, name, contentType string, invoiceID pulid.ID) error {
		content, err := s.readDocument(ctx, tenantInfo, id)
		if err != nil {
			return err
		}
		files = append(files, factoringsubmissionports.TransportFile{
			Kind:        kind,
			Name:        name,
			ContentType: contentType,
			Content:     content,
			InvoiceID:   invoiceID,
		})
		return nil
	}
	if err := read(factoringsubmissionports.TransportFileSchedulePDF, sub.SchedulePDFDocumentID,
		scheduleFileName(sub, ".pdf"), pdfContentType, pulid.Nil); err != nil {
		return nil, err
	}
	if err := read(factoringsubmissionports.TransportFileScheduleCSV, sub.ScheduleCSVDocumentID,
		scheduleFileName(sub, ".csv"), csvContentType, pulid.Nil); err != nil {
		return nil, err
	}
	if !attach {
		return files, nil
	}
	for _, item := range sub.Items {
		if item == nil {
			continue
		}
		if item.PacketDocumentID.IsNil() {
			return nil, errortypes.NewBusinessError(
				"The packet for invoice {0} has not been built yet",
				item.InvoiceNumber,
			)
		}
		if err := read(factoringsubmissionports.TransportFilePacket, item.PacketDocumentID,
			packetFileName(item), pdfContentType, item.InvoiceID); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func (s *SubmissionService) messageRenderer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) (factoringsubmissionports.MessageRenderer, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         sub.FactoringCompanyID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	format := func(minor int64) string {
		return money.FormatDecimal(sub.CurrencyCode, money.DecimalFromMinor(minor))
	}
	return func(part, parts int, attachments []string) (*factoringsubmissionports.TransportMessage, error) {
		rendered, renderErr := s.templates.RenderMessage(
			ctx,
			&documenttemplateports.RenderMessageRequest{
				TenantInfo: tenantInfo,
				Kind:       documenttemplate.KindFactoringScheduleEmail,
				Data: &factoringtemplate.FactoringScheduleEmailContext{
					FactorName:      factor.Name,
					CompanyName:     org.Name,
					Number:          sub.Number,
					ScheduleDate:    formatDate(sub.CreatedAt, org.Timezone),
					InvoiceCount:    sub.InvoiceCount,
					TotalFace:       format(sub.FaceMinor),
					ExpectedPayment: format(sub.ExpectedAdvanceMinor),
					Attachments:     attachments,
					Part:            part,
					Parts:           parts,
				},
				ReferenceID: sub.ID,
				UserID:      sub.SubmittedByID,
			},
		)
		if renderErr != nil {
			return nil, renderErr
		}
		return &factoringsubmissionports.TransportMessage{
			Subject: rendered.Subject,
			HTML:    rendered.HTML,
			Text:    rendered.Text,
		}, nil
	}, nil
}

func SendKey(sub *factoringsubmission.Submission) string {
	return submissionKeyPrefix + sub.ID.String() + scheduleSendKeyInfix +
		strconv.Itoa(sub.SendAttemptGroup)
}
