package invoicepacket

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countingPlanner struct {
	calls       atomic.Int32
	attachments []*invoice.Attachment
}

func (p *countingPlanner) PreviewApprovalInvoice(
	context.Context,
	*invoiceports.CreateInvoiceFromBillingQueueRequest,
) (*invoiceports.CreateInvoiceFromBillingQueueResult, error) {
	p.calls.Add(1)
	return &invoiceports.CreateInvoiceFromBillingQueueResult{
		Invoice: &invoice.Invoice{Attachments: p.attachments},
	}, nil
}

type draftRenderer struct{}

func (draftRenderer) RenderDraftPreview(
	context.Context,
	*invoice.Invoice,
	pagination.TenantInfo,
) (*invoiceports.InvoicePreviewResult, error) {
	return &invoiceports.InvoicePreviewResult{
		Content:  []byte("%PDF-1.7 invoice"),
		FileName: "INV-1001.pdf",
	}, nil
}

type storedDocument struct {
	doc     *document.Document
	content []byte
}

type packetDocuments struct {
	mu     sync.Mutex
	stored map[pulid.ID]storedDocument
	reads  int
}

func (d *packetDocuments) Get(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	stored, ok := d.stored[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Document not found")
	}
	return stored.doc, nil
}

func (d *packetDocuments) GetMany(
	_ context.Context,
	req documentports.GetDocumentsByIDsRequest,
) ([]*document.Document, error) {
	d.mu.Lock()
	d.reads++
	d.mu.Unlock()
	out := make([]*document.Document, 0, len(req.IDs))
	for _, id := range req.IDs {
		if stored, ok := d.stored[id]; ok {
			out = append(out, stored.doc)
		}
	}
	return out, nil
}

func (d *packetDocuments) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	stored, ok := d.stored[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("Document not found")
	}
	return &documentports.DocumentContent{
		Document: stored.doc,
		Body:     io.NopCloser(bytes.NewReader(stored.content)),
	}, nil
}

type previewHarness struct {
	service   *PreviewService
	planner   *countingPlanner
	imaging   *imagingfake.Fake
	documents *packetDocuments
}

func newPreviewHarness(t *testing.T, pages int) *previewHarness {
	t.Helper()
	imaging := imagingfake.New()
	imaging.PageCount = pages
	imaging.AssembleFunc = func(
		_ context.Context,
		_ *kernelports.ImagingAssembleRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) ([]int, error) {
		data := []byte("%PDF-1.7 packet")
		if err := yield(&kernelports.ImagingArtifact{
			ContentType: PDFContentType,
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		return []int{pages}, nil
	}
	planner := &countingPlanner{}
	documents := &packetDocuments{stored: map[pulid.ID]storedDocument{}}
	service := NewPreviewService(PreviewParams{
		Planner:   planner,
		Renderer:  draftRenderer{},
		Assembler: NewAssembler(imaging, documents, zap.NewNop()),
		Documents: documents,
		Previews:  NewPreviewCache(),
		Logger:    zap.NewNop(),
	}).(*PreviewService)
	t.Cleanup(service.warms.Wait)

	return &previewHarness{
		service:   service,
		planner:   planner,
		imaging:   imaging,
		documents: documents,
	}
}

func (h *previewHarness) carry(doc *document.Document, content string, selected bool) {
	h.documents.stored[doc.ID] = storedDocument{doc: doc, content: []byte(content)}
	h.planner.attachments = append(h.planner.attachments, &invoice.Attachment{
		DocumentID: doc.ID,
		Selected:   selected,
	})
}

func (h *previewHarness) countPagesBy(counts map[string]int) {
	h.imaging.ProbeFunc = func(
		ctx context.Context,
		req *kernelports.ImagingProbeRequest,
	) (*kernelports.ImagingProbe, error) {
		body, err := imagingfake.Read(ctx, req.Source)
		if err != nil {
			return nil, err
		}
		for marker, count := range counts {
			if strings.Contains(string(body), marker) {
				return &kernelports.ImagingProbe{
					Kind:      kernelports.ImagingKindPDF,
					PageCount: count,
				}, nil
			}
		}
		return nil, errors.New("no page count for this source")
	}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  pulid.MustNew("org_"),
		BuID:   pulid.MustNew("bu_"),
		UserID: pulid.MustNew("usr_"),
	}
}

func (h *previewHarness) preview(
	t *testing.T,
	itemID pulid.ID,
	info pagination.TenantInfo,
) *billingqueueports.PacketPreview {
	t.Helper()
	preview, err := h.service.PreviewPacket(t.Context(), &billingqueueports.PacketPreviewRequest{
		ItemID:     itemID,
		TenantInfo: info,
	})
	require.NoError(t, err)
	h.service.warms.Wait()
	return preview
}

func signedType(requires bool) *documenttype.DocumentType {
	return &documenttype.DocumentType{Name: "Proof of delivery", RequiresSignature: requires}
}

func TestPreviewingAPacketDescribesItsPagesAndBuild(t *testing.T) {
	h := newPreviewHarness(t, 3)

	preview := h.preview(t, pulid.MustNew("bqi_"), tenant())

	assert.Equal(t, "INV-1001.pdf", preview.FileName)
	assert.Equal(t, 3, preview.Pages)
	assert.Equal(t, 0, preview.Documents)
	assert.Len(t, preview.Build, 16)
	assert.Equal(t, []billingqueueports.PacketPart{
		{Kind: billingqueueports.PacketPartInvoice, FirstPage: 1, LastPage: 3},
	}, preview.Parts)
}

func TestAPacketNamesEachDocumentAndThePagesItFills(t *testing.T) {
	h := newPreviewHarness(t, 4)
	pod := &document.Document{
		ID:              pulid.MustNew("doc_"),
		OriginalName:    "pod.pdf",
		FileType:        PDFContentType,
		DocumentType:    signedType(true),
		SignatureStatus: document.SignatureStatusUnsigned,
	}
	receipt := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "lumper.jpg",
		FileType:     "image/jpeg",
	}
	left := &document.Document{
		ID:           pulid.MustNew("doc_"),
		OriginalName: "unpicked.pdf",
		FileType:     PDFContentType,
	}
	h.carry(pod, "%PDF pod", true)
	h.carry(receipt, "jpeg receipt", true)
	h.carry(left, "%PDF left", false)
	h.countPagesBy(map[string]int{"invoice": 1, "pod": 2})

	preview := h.preview(t, pulid.MustNew("bqi_"), tenant())

	assert.Equal(t, 2, preview.Documents)
	assert.Equal(t, 1, h.documents.reads)
	assert.Equal(t, []billingqueueports.PacketPart{
		{Kind: billingqueueports.PacketPartInvoice, FirstPage: 1, LastPage: 1},
		{
			Kind:       billingqueueports.PacketPartDocument,
			DocumentID: pod.ID,
			Name:       "Proof of delivery",
			FirstPage:  2,
			LastPage:   3,
			Unsigned:   true,
		},
		{
			Kind:       billingqueueports.PacketPartDocument,
			DocumentID: receipt.ID,
			Name:       "lumper.jpg",
			FirstPage:  4,
			LastPage:   4,
		},
	}, preview.Parts)
}

func TestASignedDocumentOrOneWhoseTypeNeedsNoSignatureIsNotFlagged(t *testing.T) {
	h := newPreviewHarness(t, 3)
	signed := &document.Document{
		ID:              pulid.MustNew("doc_"),
		FileType:        PDFContentType,
		DocumentType:    signedType(true),
		SignatureStatus: document.SignatureStatusSigned,
	}
	unneeded := &document.Document{
		ID:              pulid.MustNew("doc_"),
		FileType:        PDFContentType,
		DocumentType:    signedType(false),
		SignatureStatus: document.SignatureStatusUnsigned,
	}
	h.carry(signed, "%PDF signed", true)
	h.carry(unneeded, "%PDF unneeded", true)
	h.countPagesBy(map[string]int{"invoice": 1, "signed": 1, "unneeded": 1})

	preview := h.preview(t, pulid.MustNew("bqi_"), tenant())

	require.Len(t, preview.Parts, 3)
	assert.False(t, preview.Parts[1].Unsigned)
	assert.False(t, preview.Parts[2].Unsigned)
}

func TestAPacketWhosePageCountsDisagreeIsShownWithoutParts(t *testing.T) {
	h := newPreviewHarness(t, 5)
	pod := &document.Document{ID: pulid.MustNew("doc_"), FileType: PDFContentType}
	h.carry(pod, "%PDF pod", true)
	h.countPagesBy(map[string]int{"invoice": 1, "pod": 2})

	preview := h.preview(t, pulid.MustNew("bqi_"), tenant())

	assert.Equal(t, 5, preview.Pages)
	assert.Nil(t, preview.Parts)
}

func TestAPacketWithADocumentThatNoLongerExistsIsNotFound(t *testing.T) {
	h := newPreviewHarness(t, 2)
	h.planner.attachments = []*invoice.Attachment{
		{DocumentID: pulid.MustNew("doc_"), Selected: true},
	}

	_, err := h.service.PreviewPacket(t.Context(), &billingqueueports.PacketPreviewRequest{
		ItemID:     pulid.MustNew("bqi_"),
		TenantInfo: tenant(),
	})

	var notFound *errortypes.NotFoundError
	require.True(t, errors.As(err, &notFound), "got %v", err)
}

func TestOpeningThePreviewDrawsTheFirstPagesAndThumbnailsAhead(t *testing.T) {
	h := newPreviewHarness(t, 3)
	itemID := pulid.MustNew("bqi_")
	info := tenant()
	h.preview(t, itemID, info)
	renders := h.imaging.CallCount("render")
	inspections := h.imaging.CallCount("inspect_capture")

	for page := 1; page <= 3; page++ {
		image, err := h.service.PreviewPacketPage(
			t.Context(),
			&billingqueueports.PacketPageRequest{ItemID: itemID, TenantInfo: info, Page: page},
		)
		require.NoError(t, err)
		assert.Equal(t, "image/webp", image.ContentType)
		thumbnail, err := h.service.PreviewPacketThumbnail(
			t.Context(),
			&billingqueueports.PacketPageRequest{ItemID: itemID, TenantInfo: info, Page: page},
		)
		require.NoError(t, err)
		assert.Equal(t, []byte("fake-thumbnail"), thumbnail.Content)
	}

	assert.Equal(t, 2, renders, "one render for the first pages, one for the thumbnails")
	assert.Equal(t, 1, inspections)
	assert.Equal(t, renders, h.imaging.CallCount("render"))
	assert.Equal(t, inspections, h.imaging.CallCount("inspect_capture"))
	assert.Equal(t, int32(1), h.planner.calls.Load())
	assert.Equal(t, 1, h.imaging.CallCount("assemble"))
}

func TestAPacketIsDrawnEightPagesToARenderAndEachPageOnce(t *testing.T) {
	h := newPreviewHarness(t, 20)
	itemID := pulid.MustNew("bqi_")
	info := tenant()
	var asked [][]int
	h.imaging.RenderFunc = func(
		_ context.Context,
		req *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		asked = append(asked, req.Pages)
		for _, index := range req.Pages {
			data := []byte{byte(index)}
			if err := yield(&kernelports.ImagingArtifact{
				Index:       index,
				ContentType: "image/webp",
				Size:        1,
				Body:        bytes.NewReader(data),
			}); err != nil {
				return nil, err
			}
		}
		return &kernelports.ImagingRenderSummary{PageCount: 20, Rendered: len(req.Pages)}, nil
	}

	for _, page := range []int{1, 8, 3, 9, 20, 1, 17} {
		image, err := h.service.PreviewPacketPage(
			t.Context(),
			&billingqueueports.PacketPageRequest{ItemID: itemID, TenantInfo: info, Page: page},
		)
		require.NoError(t, err)
		assert.Equal(t, []byte{byte(page - 1)}, image.Content, "page %d", page)
	}

	assert.Equal(t, [][]int{
		{0, 1, 2, 3, 4, 5, 6, 7},
		{8, 9, 10, 11, 12, 13, 14, 15},
		{16, 17, 18, 19},
	}, asked)
}

func TestThumbnailsAreDrawnThirtyTwoPagesToARender(t *testing.T) {
	h := newPreviewHarness(t, 40)
	var mu sync.Mutex
	var thumbnailRenders [][]int
	h.imaging.RenderFunc = func(
		_ context.Context,
		req *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		if req.Format == kernelports.ImagingJPEG {
			mu.Lock()
			thumbnailRenders = append(thumbnailRenders, req.Pages)
			mu.Unlock()
		}
		for _, index := range req.Pages {
			if err := yield(&kernelports.ImagingArtifact{
				Index:       index,
				ContentType: "image/jpeg",
				Size:        1,
				Body:        bytes.NewReader([]byte{1}),
			}); err != nil {
				return nil, err
			}
		}
		return &kernelports.ImagingRenderSummary{PageCount: 40, Rendered: len(req.Pages)}, nil
	}

	h.preview(t, pulid.MustNew("bqi_"), tenant())

	require.Len(t, thumbnailRenders, 2)
	assert.Len(t, thumbnailRenders[0], 32)
	assert.Equal(t, []int{32, 33, 34, 35, 36, 37, 38, 39}, thumbnailRenders[1])
	assert.Equal(t, 2, h.imaging.CallCount("inspect_capture"))
}

func TestOnlyDocumentPagesWithAlmostNoInkAreBlank(t *testing.T) {
	h := newPreviewHarness(t, 4)
	pod := &document.Document{ID: pulid.MustNew("doc_"), FileType: PDFContentType}
	h.carry(pod, "%PDF pod", true)
	h.countPagesBy(map[string]int{"invoice": 1, "pod": 3})
	ink := map[int]float64{0: 0, 1: 0.2, 2: 0.001, 3: 0.004}
	h.imaging.InspectCaptureFunc = func(
		_ context.Context,
		req *kernelports.ImagingCaptureRequest,
		yield func(*kernelports.ImagingCaptureResult) error,
	) (*kernelports.ImagingCaptureSummary, error) {
		if !req.SkipBarcodes {
			return nil, errors.New("the packet index should not read barcodes")
		}
		for page, err := range req.Pages() {
			if err != nil {
				return nil, err
			}
			if err = yield(&kernelports.ImagingCaptureResult{
				Index:                page.Index,
				Thumbnail:            []byte("thumb"),
				ThumbnailContentType: "image/webp",
				InkCoverage:          ink[page.Index],
			}); err != nil {
				return nil, err
			}
		}
		return &kernelports.ImagingCaptureSummary{Inspected: req.PageCount}, nil
	}
	itemID := pulid.MustNew("bqi_")
	info := tenant()
	preview := h.preview(t, itemID, info)

	checks, err := h.service.PreviewPacketChecks(
		t.Context(),
		&billingqueueports.PacketChecksRequest{ItemID: itemID, TenantInfo: info},
	)

	require.NoError(t, err)
	assert.Equal(t, preview.Build, checks.Build)
	assert.Equal(t, []int{3}, checks.BlankPages)
}

func TestAPacketPageBuildsThePacketWhenNoPreviewHoldsIt(t *testing.T) {
	h := newPreviewHarness(t, 2)
	itemID := pulid.MustNew("bqi_")
	info := tenant()

	for range 2 {
		_, err := h.service.PreviewPacketPage(
			t.Context(),
			&billingqueueports.PacketPageRequest{ItemID: itemID, TenantInfo: info, Page: 2},
		)
		require.NoError(t, err)
	}

	assert.Equal(t, int32(1), h.planner.calls.Load())
	assert.Equal(t, 1, h.imaging.CallCount("assemble"))
}

func TestReopeningThePreviewRebuildsThePacket(t *testing.T) {
	h := newPreviewHarness(t, 1)
	itemID := pulid.MustNew("bqi_")
	info := tenant()

	h.preview(t, itemID, info)
	h.preview(t, itemID, info)

	assert.Equal(t, int32(2), h.planner.calls.Load())
}

func TestAnotherTenantNeverReadsAPacketBuiltForThisOne(t *testing.T) {
	h := newPreviewHarness(t, 1)
	itemID := pulid.MustNew("bqi_")
	h.preview(t, itemID, tenant())

	_, err := h.service.PreviewPacketPage(
		t.Context(),
		&billingqueueports.PacketPageRequest{ItemID: itemID, TenantInfo: tenant(), Page: 1},
	)
	require.NoError(t, err)

	assert.Equal(t, int32(2), h.planner.calls.Load())
}

func TestAPagePastTheEndOfThePacketIsNotFound(t *testing.T) {
	h := newPreviewHarness(t, 2)

	for _, draw := range []func(
		context.Context,
		*billingqueueports.PacketPageRequest,
	) (*billingqueueports.PacketPage, error){
		h.service.PreviewPacketPage,
		h.service.PreviewPacketThumbnail,
	} {
		_, err := draw(t.Context(), &billingqueueports.PacketPageRequest{
			ItemID:     pulid.MustNew("bqi_"),
			TenantInfo: tenant(),
			Page:       3,
		})

		var notFound *errortypes.NotFoundError
		require.True(t, errors.As(err, &notFound), "got %v", err)
	}
	assert.Equal(t, 0, h.imaging.CallCount("render"))
}

func TestPageZeroIsRefused(t *testing.T) {
	h := newPreviewHarness(t, 2)

	_, err := h.service.PreviewPacketPage(
		t.Context(),
		&billingqueueports.PacketPageRequest{
			ItemID:     pulid.MustNew("bqi_"),
			TenantInfo: tenant(),
			Page:       0,
		},
	)

	var invalid *errortypes.Error
	require.True(t, errors.As(err, &invalid), "got %v", err)
	assert.Equal(t, int32(0), h.planner.calls.Load())
}
