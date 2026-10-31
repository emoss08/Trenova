package invoicepacket

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// ApprovalPlanner builds, without saving it, the invoice approving an item makes.
type ApprovalPlanner interface {
	PreviewApprovalInvoice(
		ctx context.Context,
		req *invoiceports.CreateInvoiceFromBillingQueueRequest,
	) (*invoiceports.CreateInvoiceFromBillingQueueResult, error)
}

// DraftRenderer renders an invoice that has not been saved.
type DraftRenderer interface {
	RenderDraftPreview(
		ctx context.Context,
		entity *invoice.Invoice,
		tenantInfo pagination.TenantInfo,
	) (*invoiceports.InvoicePreviewResult, error)
}

type PreviewService struct {
	planner   ApprovalPlanner
	renderer  DraftRenderer
	assembler *Assembler
	documents documentports.InvoiceDocumentService
}

var _ billingqueueports.PacketPreviewer = (*PreviewService)(nil)

func NewPreviewService(
	planner ApprovalPlanner,
	renderer DraftRenderer,
	assembler *Assembler,
	documents documentports.InvoiceDocumentService,
) billingqueueports.PacketPreviewer {
	return &PreviewService{
		planner:   planner,
		renderer:  renderer,
		assembler: assembler,
		documents: documents,
	}
}

// PreviewPacket is the packet approving the item would send, built by the same
// planning approval runs, so what the biller sees is what the customer gets.
func (s *PreviewService) PreviewPacket(
	ctx context.Context,
	req *billingqueueports.PacketPreviewRequest,
) (*billingqueueports.PacketPreview, error) {
	planned, err := s.planner.PreviewApprovalInvoice(
		ctx,
		&invoiceports.CreateInvoiceFromBillingQueueRequest{
			BillingQueueItemID: req.ItemID,
			TenantInfo:         req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if planned.DeferredToStatement || planned.Invoice == nil {
		return nil, errortypes.NewBusinessError(
			"This customer is billed on a statement; its packet is built when the statement closes",
		)
	}

	rendered, err := s.renderer.RenderDraftPreview(ctx, planned.Invoice, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	parts := []imagingkit.Part{{Source: kernelports.ImagingBytes(PDFContentType, rendered.Content)}}

	carried := 0
	for _, attachment := range planned.Invoice.Attachments {
		if attachment == nil || !attachment.Selected {
			continue
		}
		doc, getErr := s.documents.Get(ctx, documentports.GetDocumentByIDRequest{
			ID:         attachment.DocumentID,
			TenantInfo: req.TenantInfo,
		})
		if getErr != nil {
			return nil, getErr
		}
		part, partErr := s.assembler.Part(ctx, req.TenantInfo, doc)
		if partErr != nil {
			return nil, errortypes.NewBusinessError(
				"{0} could not be added to the packet: {1}",
				doc.OriginalName,
				partErr.Error(),
			)
		}
		if part != nil {
			parts = append(parts, *part)
			carried++
		}
	}

	packet, err := s.assembler.Merge(ctx, parts, kernelports.ImagingInteractive)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.PacketPreview{
		FileName:  rendered.FileName,
		Content:   packet,
		Documents: carried,
	}, nil
}
