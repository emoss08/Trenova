package invoicepacket

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"
)

const (
	maxPartReads  = 4
	maxPartProbes = 4
)

type packetSource struct {
	part imagingkit.Part
	meta billingqueueports.PacketPart
}

func (s *PreviewService) build(
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
	documents, err := s.packetDocuments(ctx, req.TenantInfo, planned.Invoice.Attachments)
	if err != nil {
		return nil, err
	}

	sources := make([]packetSource, 0, len(documents)+1)
	sources = append(sources, packetSource{
		part: imagingkit.Part{Source: kernelports.ImagingBytes(PDFContentType, rendered.Content)},
		meta: billingqueueports.PacketPart{Kind: billingqueueports.PacketPartInvoice},
	})
	carried, err := s.readParts(
		ctx,
		req.TenantInfo,
		documents,
		AttachmentRotations(planned.Invoice.Attachments),
	)
	if err != nil {
		return nil, err
	}
	sources = append(sources, carried...)

	packet, pages, counts, err := s.assemble(ctx, sources)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(packet)

	return &billingqueueports.PacketPreview{
		FileName:  rendered.FileName,
		Content:   packet,
		Pages:     pages,
		Documents: len(carried),
		Build:     hex.EncodeToString(digest[:8]),
		Parts:     partsOf(sources, counts, pages),
	}, nil
}

func (s *PreviewService) packetDocuments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	attachments []*invoice.Attachment,
) ([]*document.Document, error) {
	ids := make([]pulid.ID, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment != nil && attachment.Selected {
			ids = append(ids, attachment.DocumentID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	found, err := s.documents.GetMany(ctx, documentports.GetDocumentsByIDsRequest{
		IDs:                 ids,
		TenantInfo:          tenantInfo,
		IncludeDocumentType: true,
	})
	if err != nil {
		return nil, err
	}
	byID := make(map[pulid.ID]*document.Document, len(found))
	for _, doc := range found {
		byID[doc.ID] = doc
	}
	ordered := make([]*document.Document, 0, len(ids))
	for _, id := range ids {
		doc, ok := byID[id]
		if !ok {
			return nil, errortypes.NewNotFoundError("A document in this packet no longer exists")
		}
		ordered = append(ordered, doc)
	}

	return ordered, nil
}

func (s *PreviewService) readParts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documents []*document.Document,
	rotations map[pulid.ID][]billingvocab.PageRotation,
) ([]packetSource, error) {
	parts := make([]*imagingkit.Part, len(documents))
	reads := pool.New().WithContext(ctx).WithCancelOnError().WithMaxGoroutines(maxPartReads)
	for i, doc := range documents {
		reads.Go(func(ctx context.Context) error {
			part, err := s.assembler.Part(ctx, PartRequest{
				TenantInfo: tenantInfo,
				Document:   doc,
				Rotations:  rotations[doc.ID],
				Priority:   kernelports.ImagingInteractive,
			})
			if err != nil {
				return errortypes.NewBusinessError(
					"{0} could not be added to the packet: {1}",
					doc.OriginalName,
					err.Error(),
				)
			}
			parts[i] = part
			return nil
		})
	}
	if err := reads.Wait(); err != nil {
		return nil, err
	}

	out := make([]packetSource, 0, len(documents))
	for i, doc := range documents {
		if parts[i] == nil {
			continue
		}
		out = append(out, packetSource{
			part: *parts[i],
			meta: billingqueueports.PacketPart{
				Kind:       billingqueueports.PacketPartDocument,
				DocumentID: doc.ID,
				Name:       documentName(doc),
				Unsigned:   documentUnsigned(doc),
			},
		})
	}

	return out, nil
}

func (s *PreviewService) assemble(
	ctx context.Context,
	sources []packetSource,
) (packet []byte, pages int, counts []int, err error) {
	parts := make([]imagingkit.Part, len(sources))
	counts = make([]int, len(sources))
	work := pool.New().WithContext(ctx).WithCancelOnError().WithMaxGoroutines(maxPartProbes + 1)
	work.Go(func(ctx context.Context) error {
		for i := range sources {
			parts[i] = sources[i].part
		}
		var mergeErr error
		packet, pages, mergeErr = s.assembler.Merge(ctx, parts, kernelports.ImagingInteractive)
		return mergeErr
	})
	for i := range sources {
		source := sources[i].part.Source
		if !strings.EqualFold(source.ContentType, PDFContentType) {
			counts[i] = 1
			continue
		}
		work.Go(func(ctx context.Context) error {
			count, probeErr := imagingkit.PageCount(
				ctx,
				s.assembler.imaging,
				source,
				kernelports.ImagingInteractive,
			)
			if probeErr != nil {
				s.l.Warn(
					"could not count a packet part's pages; the viewer shows the packet without its parts",
					zap.Error(probeErr),
				)
				counts[i] = 0
				return nil
			}
			counts[i] = count
			return nil
		})
	}
	if err = work.Wait(); err != nil {
		return nil, 0, nil, err
	}

	return packet, pages, counts, nil
}

func partsOf(sources []packetSource, counts []int, pages int) []billingqueueports.PacketPart {
	parts := make([]billingqueueports.PacketPart, 0, len(sources))
	next := 1
	for i, source := range sources {
		if counts[i] < 1 {
			return nil
		}
		part := source.meta
		part.FirstPage = next
		part.LastPage = next + counts[i] - 1
		next = part.LastPage + 1
		parts = append(parts, part)
	}
	if next-1 != pages {
		return nil
	}

	return parts
}

func documentName(doc *document.Document) string {
	if doc.DocumentType != nil && strings.TrimSpace(doc.DocumentType.Name) != "" {
		return doc.DocumentType.Name
	}
	return doc.OriginalName
}

func documentUnsigned(doc *document.Document) bool {
	return doc.DocumentType != nil &&
		doc.DocumentType.RequiresSignature &&
		doc.SignatureStatus == document.SignatureStatusUnsigned
}

func AttachmentRotations(
	attachments []*invoice.Attachment,
) map[pulid.ID][]billingvocab.PageRotation {
	out := make(map[pulid.ID][]billingvocab.PageRotation, len(attachments))
	for _, attachment := range attachments {
		if attachment != nil && len(attachment.PageRotations) > 0 {
			out[attachment.DocumentID] = attachment.PageRotations
		}
	}
	return out
}
