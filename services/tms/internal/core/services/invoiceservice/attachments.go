package invoiceservice

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/encodingutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (s *Service) documentForID(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*document.Document, error) {
	if s.documentService == nil {
		return nil, errortypes.NewBusinessError("Document service is not configured")
	}
	return s.documentService.Get(
		ctx,
		repositories.GetDocumentByIDRequest{ID: documentID, TenantInfo: tenantInfo},
	)
}

func (s *Service) emailAttachmentsForPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planned []*servicesports.InvoiceSendPlanAttachment,
) ([]servicesports.EmailAttachment, error) {
	result := make([]servicesports.EmailAttachment, 0, len(planned))
	for _, item := range planned {
		content, err := s.documentService.GetDownloadContent(
			ctx,
			repositories.GetDocumentByIDRequest{
				ID:         item.DocumentID,
				TenantInfo: tenantInfo,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("prepare attachment %s: %w", item.FileName, err)
		}
		body, err := io.ReadAll(content.Body)
		closeErr := content.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read attachment %s: %w", item.FileName, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close attachment %s: %w", item.FileName, closeErr)
		}
		result = append(result, servicesports.EmailAttachment{
			FileName:    item.FileName,
			ContentType: item.ContentType,
			Content:     body,
			SizeBytes:   int64(len(body)),
		})
	}
	return result, nil
}

func readDocumentBody(body io.ReadCloser, label string) ([]byte, error) {
	content, err := io.ReadAll(body)
	closeErr := body.Close()
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to read {0}", label).WithInternal(err)
	}
	if closeErr != nil {
		return nil, errortypes.NewDatabaseError("Failed to close {0}", label).WithInternal(closeErr)
	}
	return content, nil
}

func (s *Service) materializePartLinks(
	ctx context.Context,
	req *servicesports.InvoiceSendRequest,
	actor *servicesports.RequestActor,
	part *servicesports.InvoiceSendPlanPart,
) (string, []*invoice.EmailAttemptAttachment, error) {
	body := strings.TrimSpace(partBodyBase(part))
	linked := make([]*invoice.EmailAttemptAttachment, 0, len(part.Links))
	for _, link := range part.Links {
		rawToken, hash, err := newShareToken()
		if err != nil {
			return body, linked, err
		}
		share, err := s.repo.CreateDocumentShareToken(ctx, &invoice.DocumentShareToken{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			InvoiceID:      req.InvoiceID,
			DocumentID:     link.DocumentID,
			TokenHash:      hash,
			ExpiresAt:      time.Now().Add(shareTokenTTL).Unix(),
			CreatedByID:    actorUserID(actor, req.TenantInfo),
		})
		if err != nil {
			return body, linked, err
		}
		link.URL = signedDocumentURL(s.linkBaseURL(req.BaseURL), rawToken)
		body += "\n\nSupporting document link: " + link.FileName + "\n" + link.URL
		linked = append(linked, &invoice.EmailAttemptAttachment{
			DocumentID:   link.DocumentID,
			FileName:     link.FileName,
			SizeBytes:    link.SizeBytes,
			EncodedBytes: encodingutils.EncodedBase64Size(link.SizeBytes),
			Method:       invoice.AttachmentDeliveryMethodLink,
			ShareTokenID: share.ID,
			Reason:       link.Reason,
		})
	}
	return body, linked, nil
}

func attemptAttachmentsForPlan(
	planned []*servicesports.InvoiceSendPlanAttachment,
	linked []*invoice.EmailAttemptAttachment,
) []*invoice.EmailAttemptAttachment {
	result := make([]*invoice.EmailAttemptAttachment, 0, len(planned)+len(linked))
	for _, item := range planned {
		result = append(result, &invoice.EmailAttemptAttachment{
			DocumentID:   item.DocumentID,
			FileName:     item.FileName,
			ContentType:  item.ContentType,
			SizeBytes:    item.SizeBytes,
			EncodedBytes: item.EncodedBytes,
			Method:       invoice.AttachmentDeliveryMethodAttached,
		})
	}
	result = append(result, linked...)
	return result
}

func attemptWarnings(
	plan *servicesports.InvoiceSendPlan,
	part *servicesports.InvoiceSendPlanPart,
) []string {
	capacity := len(plan.Warnings) + len(part.Warnings)
	if plan.OpenTracking {
		capacity++
	}
	if len(plan.Headers) > 0 {
		capacity++
	}
	result := make([]string, 0, capacity)
	result = append(result, plan.Warnings...)
	result = append(result, part.Warnings...)
	if plan.OpenTracking {
		result = append(result, "Read receipt/open tracking requested")
	}
	if len(plan.Headers) > 0 {
		result = append(result, "Custom email headers requested")
	}
	return sliceutils.DedupeSorted(result)
}

func planAttachment(
	doc *document.Document,
	invoicePDF bool,
) *servicesports.InvoiceSendPlanAttachment {
	return &servicesports.InvoiceSendPlanAttachment{
		DocumentID:   doc.ID,
		FileName:     doc.OriginalName,
		ContentType:  doc.FileType,
		SizeBytes:    doc.FileSize,
		EncodedBytes: encodingutils.EncodedBase64Size(doc.FileSize),
		InvoicePDF:   invoicePDF,
	}
}

func providerLimit(profile *email.Profile) int64 {
	if profile != nil && profile.Provider == email.ProviderResend {
		return resendMessageLimitBytes
	}
	return postmarkMessageLimitBytes
}
