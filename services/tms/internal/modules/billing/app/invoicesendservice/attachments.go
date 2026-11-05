package invoicesendservice

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepacket"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
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
	return s.documentService.Get(
		ctx,
		documentports.GetDocumentByIDRequest{ID: documentID, TenantInfo: tenantInfo},
	)
}

func (s *Service) emailAttachmentsForPlan(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	planned []*invoiceports.InvoiceSendPlanAttachment,
) ([]emailports.EmailAttachment, error) {
	result := make([]emailports.EmailAttachment, 0, len(planned))
	for _, item := range planned {
		if len(item.PageRotations) > 0 && !item.InvoicePDF {
			turned, err := s.turnedAttachment(ctx, tenantInfo, item)
			if err != nil {
				return nil, fmt.Errorf("turn the pages of attachment %s: %w", item.FileName, err)
			}
			if turned != nil {
				result = append(result, *turned)
				continue
			}
		}
		content, err := s.documentService.GetDownloadContent(
			ctx,
			documentports.GetDocumentByIDRequest{
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
		result = append(result, emailports.EmailAttachment{
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
	req *invoiceports.InvoiceSendRequest,
	actor *permissionports.RequestActor,
	part *invoiceports.InvoiceSendPlanPart,
) (string, []*invoice.EmailAttemptAttachment, error) {
	body := strings.TrimSpace(partBodyBase(part))
	linked := make([]*invoice.EmailAttemptAttachment, 0, len(part.Links))
	if len(part.Links) > 0 && s.webBaseURL == "" {
		return body, linked, errortypes.NewBusinessError(
			"Document links need a public base URL; set app.webBaseUrl",
		)
	}
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
			CreatedByID:    invoiceaction.ActorUserID(actor, req.TenantInfo),
		})
		if err != nil {
			return body, linked, err
		}
		link.URL = signedDocumentURL(s.webBaseURL, rawToken)
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
	planned []*invoiceports.InvoiceSendPlanAttachment,
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
	plan *invoiceports.InvoiceSendPlan,
	part *invoiceports.InvoiceSendPlanPart,
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
) *invoiceports.InvoiceSendPlanAttachment {
	return &invoiceports.InvoiceSendPlanAttachment{
		DocumentID:   doc.ID,
		FileName:     doc.OriginalName,
		ContentType:  doc.FileType,
		SizeBytes:    doc.FileSize,
		EncodedBytes: encodingutils.EncodedBase64Size(doc.FileSize),
		InvoicePDF:   invoicePDF,
	}
}

func providerLimit(profile *email.Profile) int64 {
	if profile == nil {
		return emailports.PostmarkMessageLimitBytes
	}
	return emailports.MessageLimitBytes(profile.Provider)
}

func (s *Service) turnedAttachment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	item *invoiceports.InvoiceSendPlanAttachment,
) (*emailports.EmailAttachment, error) {
	doc, err := s.documentService.Get(
		ctx,
		documentports.GetDocumentByIDRequest{ID: item.DocumentID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}
	part, err := s.packets.Part(ctx, invoicepacket.PartRequest{
		TenantInfo: tenantInfo,
		Document:   doc,
		Rotations:  item.PageRotations,
		Priority:   kernelports.ImagingBatch,
	})
	if err != nil || part == nil {
		return nil, err
	}
	turned, _, err := s.packets.Merge(ctx, []imagingkit.Part{*part}, kernelports.ImagingBatch)
	if err != nil {
		return nil, err
	}

	return &emailports.EmailAttachment{
		FileName:    strings.TrimSuffix(item.FileName, filepath.Ext(item.FileName)) + ".pdf",
		ContentType: invoicepacket.PDFContentType,
		Content:     turned,
		SizeBytes:   int64(len(turned)),
	}, nil
}
