package invoicepacket

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	maxWarming    = 8
	warmTimeout   = 2 * time.Minute
	pageChunkSize = 8
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
	previews  PreviewCache
	warming   *semaphore.Weighted
	warms     sync.WaitGroup
	l         *zap.Logger
}

var _ billingqueueports.PacketPreviewer = (*PreviewService)(nil)

type PreviewParams struct {
	Planner   ApprovalPlanner
	Renderer  DraftRenderer
	Assembler *Assembler
	Documents documentports.InvoiceDocumentService
	Previews  PreviewCache
	Logger    *zap.Logger
}

func NewPreviewService(p PreviewParams) billingqueueports.PacketPreviewer {
	return &PreviewService{
		planner:   p.Planner,
		renderer:  p.Renderer,
		assembler: p.Assembler,
		documents: p.Documents,
		previews:  p.Previews,
		warming:   semaphore.NewWeighted(maxWarming),
		l:         p.Logger.Named("invoice-packet-preview"),
	}
}

// PreviewPacket is the packet approving the item would send, built by the same
// planning approval runs, so what the biller sees is what the customer gets.
func (s *PreviewService) PreviewPacket(
	ctx context.Context,
	req *billingqueueports.PacketPreviewRequest,
) (*billingqueueports.PacketPreview, error) {
	preview, err := s.build(ctx, req)
	if err != nil {
		return nil, err
	}
	key := previewKeyOf(req.ItemID, req.TenantInfo)
	s.previews.Put(key, preview)
	s.warm(ctx, key, preview)

	return preview, nil
}

func (s *PreviewService) PreviewPacketPage(
	ctx context.Context,
	req *billingqueueports.PacketPageRequest,
) (*billingqueueports.PacketPage, error) {
	key, preview, err := s.pageOf(ctx, req)
	if err != nil {
		return nil, err
	}
	chunk := (req.Page - 1) / pageChunkSize
	pages, err := s.chunk(ctx, key, preview, chunk)
	if errors.Is(err, kernelports.ErrImagingPageOutOfRange) {
		return nil, noSuchPage(req.Page)
	}
	if err != nil {
		return nil, err
	}
	at := req.Page - 1 - chunk*pageChunkSize
	if at >= len(pages) || pages[at].Content == nil {
		return nil, noSuchPage(req.Page)
	}

	return &pages[at], nil
}

func (s *PreviewService) PreviewPacketThumbnail(
	ctx context.Context,
	req *billingqueueports.PacketPageRequest,
) (*billingqueueports.PacketPage, error) {
	key, preview, err := s.pageOf(ctx, req)
	if err != nil {
		return nil, err
	}
	index, err := s.index(ctx, key, preview)
	if err != nil {
		return nil, err
	}
	thumbnail := index.Thumbnails[req.Page-1]
	if thumbnail.Content == nil {
		return nil, noSuchPage(req.Page)
	}

	return &thumbnail, nil
}

func (s *PreviewService) PreviewPacketChecks(
	ctx context.Context,
	req *billingqueueports.PacketChecksRequest,
) (*billingqueueports.PacketChecks, error) {
	key := previewKeyOf(req.ItemID, req.TenantInfo)
	preview, err := s.load(ctx, key, req.ItemID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	index, err := s.index(ctx, key, preview)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.PacketChecks{
		Build:      preview.Build,
		BlankPages: blankDocumentPages(preview.Parts, index.Ink),
	}, nil
}

func blankDocumentPages(parts []billingqueueports.PacketPart, ink []float64) []int {
	blank := make([]int, 0)
	for _, part := range parts {
		if part.Kind != billingqueueports.PacketPartDocument {
			continue
		}
		for page := part.FirstPage; page <= part.LastPage && page <= len(ink); page++ {
			coverage := ink[page-1]
			if coverage >= 0 && coverage < capture.BlankThreshold {
				blank = append(blank, page)
			}
		}
	}

	return blank
}

func noSuchPage(page int) error {
	return errortypes.NewNotFoundError("The packet has no page {0}", page)
}

func previewKeyOf(itemID pulid.ID, tenantInfo pagination.TenantInfo) PreviewKey {
	return PreviewKey{OrgID: tenantInfo.OrgID, BuID: tenantInfo.BuID, ItemID: itemID}
}

func (s *PreviewService) pageOf(
	ctx context.Context,
	req *billingqueueports.PacketPageRequest,
) (PreviewKey, *billingqueueports.PacketPreview, error) {
	key := previewKeyOf(req.ItemID, req.TenantInfo)
	if req.Page < 1 {
		return key, nil, errortypes.NewValidationError(
			"page",
			errortypes.ErrInvalid,
			"Page number must be a whole number from 1",
		)
	}
	preview, err := s.load(ctx, key, req.ItemID, req.TenantInfo)
	if err != nil {
		return key, nil, err
	}
	if req.Page > preview.Pages {
		return key, nil, noSuchPage(req.Page)
	}

	return key, preview, nil
}

func (s *PreviewService) load(
	ctx context.Context,
	key PreviewKey,
	itemID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.PacketPreview, error) {
	return s.previews.Load(
		ctx,
		key,
		func(ctx context.Context) (*billingqueueports.PacketPreview, error) {
			return s.build(ctx, &billingqueueports.PacketPreviewRequest{
				ItemID:     itemID,
				TenantInfo: tenantInfo,
			})
		},
	)
}

func (s *PreviewService) chunk(
	ctx context.Context,
	key PreviewKey,
	preview *billingqueueports.PacketPreview,
	chunk int,
) ([]billingqueueports.PacketPage, error) {
	first := chunk*pageChunkSize + 1
	last := min(first+pageChunkSize-1, preview.Pages)

	return s.previews.LoadPages(
		ctx,
		PageChunkKey{Preview: key, Build: preview.Build, Chunk: chunk},
		func(ctx context.Context) ([]billingqueueports.PacketPage, error) {
			return s.assembler.RenderPages(ctx, preview.Content, first, last)
		},
	)
}

func (s *PreviewService) index(
	ctx context.Context,
	key PreviewKey,
	preview *billingqueueports.PacketPreview,
) (*PageIndex, error) {
	return s.previews.LoadIndex(
		ctx,
		PageIndexKey{Preview: key, Build: preview.Build},
		func(ctx context.Context) (*PageIndex, error) {
			return s.assembler.IndexPages(ctx, preview.Content, preview.Pages)
		},
	)
}

func (s *PreviewService) warm(
	ctx context.Context,
	key PreviewKey,
	preview *billingqueueports.PacketPreview,
) {
	if !s.warming.TryAcquire(1) {
		return
	}
	s.warms.Go(func() {
		defer s.warming.Release(1)
		warmCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), warmTimeout)
		defer cancel()
		if _, err := s.chunk(warmCtx, key, preview, 0); err != nil {
			s.l.Warn("could not draw a packet's first pages ahead of the viewer",
				zap.String("billingQueueItemId", key.ItemID.String()), zap.Error(err))
		}
		if _, err := s.index(warmCtx, key, preview); err != nil {
			s.l.Warn("could not index a packet's pages ahead of the viewer",
				zap.String("billingQueueItemId", key.ItemID.String()), zap.Error(err))
		}
	})
}
