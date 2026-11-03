package invoicepacket

import (
	"context"
	"io"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	PDFContentType  = "application/pdf"
	jpegContentType = "image/jpeg"
	pngContentType  = "image/png"
	MaxPacketBytes  = 256 << 20
	MaxSourceBytes  = 50 << 20

	pageImageWidth    = 1400
	pageImageHeight   = 1800
	pageImageQuality  = 82
	maxPageImageBytes = 32 << 20
)

// Assembler turns an invoice's PDF and its documents into the one PDF its
// packet goes out as. Factoring submissions and the billing queue's packet
// preview both build packets through it.
type Assembler struct {
	imaging   kernelports.Imaging
	documents documentports.InvoiceDocumentService
	l         *zap.Logger
}

func NewAssembler(
	imaging kernelports.Imaging,
	documents documentports.InvoiceDocumentService,
	logger *zap.Logger,
) *Assembler {
	return &Assembler{imaging: imaging, documents: documents, l: logger.Named("invoice-packet")}
}

// Read is a stored document's bytes, refused past MaxSourceBytes.
func (a *Assembler) Read(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) ([]byte, error) {
	content, err := a.documents.GetDownloadContent(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	defer content.Body.Close()
	body, err := io.ReadAll(io.LimitReader(content.Body, MaxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxSourceBytes {
		return nil, errortypes.NewBusinessError("A document is larger than 50 MB")
	}

	return body, nil
}

type PartRequest struct {
	TenantInfo pagination.TenantInfo
	Document   *document.Document
	Rotations  []billingvocab.PageRotation
	Priority   kernelports.ImagingPriority
}

// Part is a document as pages of a packet, each page turned as the biller
// turned it. A file that is not a PDF or an image cannot be drawn into one and
// is left out.
func (a *Assembler) Part(ctx context.Context, req PartRequest) (*imagingkit.Part, error) {
	tenantInfo, doc := req.TenantInfo, req.Document
	contentType := strings.ToLower(strings.TrimSpace(doc.FileType))
	switch contentType {
	case PDFContentType, jpegContentType, pngContentType:
	default:
		a.l.Info("left a document out of a packet because it is not a PDF or image",
			zap.String("documentId", doc.ID.String()), zap.String("fileType", doc.FileType))
		return nil, nil //nolint:nilnil // a document a packet cannot carry is skipped, not an error
	}
	content, err := a.Read(ctx, tenantInfo, doc.ID)
	if err != nil {
		return nil, err
	}

	part := &imagingkit.Part{Source: kernelports.ImagingBytes(contentType, content)}
	if len(req.Rotations) == 0 {
		return part, nil
	}
	if contentType != PDFContentType {
		part.Rotation = rotationOfPage(req.Rotations, 1)
		return part, nil
	}
	pages, err := imagingkit.PageCount(ctx, a.imaging, part.Source, req.Priority)
	if err != nil {
		return nil, err
	}
	part.Spans = turnedSpans(req.Rotations, pages)

	return part, nil
}

func rotationOfPage(rotations []billingvocab.PageRotation, page int) int {
	for _, turn := range rotations {
		if turn.Page == page {
			return turn.Rotation
		}
	}
	return 0
}

func turnedSpans(rotations []billingvocab.PageRotation, pages int) []imagingkit.Span {
	turns := make(map[int]int, len(rotations))
	for _, turn := range rotations {
		if turn.Page >= 1 && turn.Page <= pages && turn.Rotation != 0 {
			turns[turn.Page] = turn.Rotation
		}
	}
	if len(turns) == 0 {
		return nil
	}
	spans := make([]imagingkit.Span, 0, len(turns)*2+1)
	for page := 1; page <= pages; page++ {
		rotation := turns[page]
		if last := len(spans) - 1; last >= 0 && spans[last].Rotation == rotation {
			spans[last].Last = page - 1
			continue
		}
		spans = append(spans, imagingkit.Span{First: page - 1, Last: page - 1, Rotation: rotation})
	}

	return spans
}

func (a *Assembler) Merge(
	ctx context.Context,
	parts []imagingkit.Part,
	priority kernelports.ImagingPriority,
) (packet []byte, pages int, err error) {
	return imagingkit.Merge(ctx, a.imaging, parts, MaxPacketBytes, priority)
}

func (a *Assembler) RenderPages(
	ctx context.Context,
	packet []byte,
	first, last int,
) ([]billingqueueports.PacketPage, error) {
	indexes := make([]int, 0, last-first+1)
	for page := first; page <= last; page++ {
		indexes = append(indexes, page-1)
	}
	pages := make([]billingqueueports.PacketPage, len(indexes))
	_, err := a.imaging.Render(ctx, &kernelports.ImagingRenderRequest{
		Source:    kernelports.ImagingBytes(PDFContentType, packet),
		Pages:     indexes,
		FitWidth:  pageImageWidth,
		FitHeight: pageImageHeight,
		Format:    kernelports.ImagingWebP,
		Quality:   pageImageQuality,
		Priority:  kernelports.ImagingInteractive,
	}, func(artifact *kernelports.ImagingArtifact) error {
		at := artifact.Index - (first - 1)
		if at < 0 || at >= len(pages) {
			return nil
		}
		data, collectErr := imagingkit.Collect(artifact, maxPageImageBytes)
		if collectErr != nil {
			return collectErr
		}
		pages[at] = billingqueueports.PacketPage{Content: data, ContentType: artifact.ContentType}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return pages, nil
}

// RequiredDocumentIDs are the documents an invoice carries when the biller picked
// none: every document that satisfies one of the customer's required types, in
// the order the requirements list them.
func RequiredDocumentIDs(readiness *shipmentbillingports.ShipmentBillingReadiness) []pulid.ID {
	if readiness == nil {
		return nil
	}
	out := make([]pulid.ID, 0)
	for _, requirement := range readiness.Requirements {
		for _, raw := range requirement.DocumentIDs {
			id, err := pulid.Parse(raw)
			if err != nil || slices.Contains(out, id) {
				continue
			}
			out = append(out, id)
		}
	}

	return out
}
