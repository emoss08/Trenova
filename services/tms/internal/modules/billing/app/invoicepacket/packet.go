package invoicepacket

import (
	"context"
	"io"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
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

// Part is a document as pages of a packet. A file that is not a PDF or an image
// cannot be drawn into one and is left out.
func (a *Assembler) Part(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
) (*imagingkit.Part, error) {
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

	return &imagingkit.Part{Source: kernelports.ImagingBytes(contentType, content)}, nil
}

func (a *Assembler) Merge(
	ctx context.Context,
	parts []imagingkit.Part,
	priority kernelports.ImagingPriority,
) ([]byte, error) {
	packet, _, err := imagingkit.Merge(ctx, a.imaging, parts, MaxPacketBytes, priority)
	return packet, err
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
