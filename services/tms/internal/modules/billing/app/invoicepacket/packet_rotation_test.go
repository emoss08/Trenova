package invoicepacket

import (
	"bytes"
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func turns(pairs ...int) []billingvocab.PageRotation {
	out := make([]billingvocab.PageRotation, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, billingvocab.PageRotation{Page: pairs[i], Rotation: pairs[i+1]})
	}
	return out
}

func TestTurnedSpansCoverEveryPageAndTurnOnlyTheTurnedOnes(t *testing.T) {
	spans := turnedSpans(turns(2, 90, 3, 90, 5, 180, 9, 90, 4, 0), 5)

	assert.Equal(t, []imagingkit.Span{
		{First: 0, Last: 0, Rotation: 0},
		{First: 1, Last: 2, Rotation: 90},
		{First: 3, Last: 3, Rotation: 0},
		{First: 4, Last: 4, Rotation: 180},
	}, spans)
}

func TestTurnsPastTheLastPageLeaveTheDocumentWhole(t *testing.T) {
	assert.Nil(t, turnedSpans(turns(7, 90), 3))
}

func TestAnUnturnedDocumentIsNotProbed(t *testing.T) {
	h := newPreviewHarness(t, 1)
	doc := &document.Document{ID: pulid.MustNew("doc_"), FileType: PDFContentType}
	h.documents.stored[doc.ID] = storedDocument{doc: doc, content: []byte("%PDF doc")}

	part, err := h.service.assembler.Part(
		t.Context(),
		PartRequest{TenantInfo: tenant(), Document: doc},
	)

	require.NoError(t, err)
	assert.Nil(t, part.Spans)
	assert.Equal(t, 0, h.imaging.CallCount("probe"))
}

func TestATurnedImageTurnsAsOnePage(t *testing.T) {
	h := newPreviewHarness(t, 1)
	doc := &document.Document{ID: pulid.MustNew("doc_"), FileType: "image/jpeg"}
	h.documents.stored[doc.ID] = storedDocument{doc: doc, content: []byte("jpeg")}

	part, err := h.service.assembler.Part(t.Context(), PartRequest{
		TenantInfo: tenant(),
		Document:   doc,
		Rotations:  turns(1, 270),
	})

	require.NoError(t, err)
	assert.Equal(t, 270, part.Rotation)
	assert.Nil(t, part.Spans)
	assert.Equal(t, 0, h.imaging.CallCount("probe"))
}

func TestThePreviewAssemblesTheTurnsTheInvoiceCarries(t *testing.T) {
	h := newPreviewHarness(t, 3)
	pod := &document.Document{ID: pulid.MustNew("doc_"), FileType: PDFContentType}
	h.carry(pod, "%PDF pod", true)
	h.planner.attachments[0].PageRotations = turns(2, 90)
	h.countPagesBy(map[string]int{"invoice": 1, "pod": 2})
	var ranges []kernelports.ImagingPageRange
	h.imaging.AssembleFunc = func(
		_ context.Context,
		req *kernelports.ImagingAssembleRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) ([]int, error) {
		ranges = req.Outputs[0]
		data := []byte("%PDF-1.7 packet")
		if err := yield(&kernelports.ImagingArtifact{
			ContentType: PDFContentType,
			Size:        int64(len(data)),
			Body:        bytes.NewReader(data),
		}); err != nil {
			return nil, err
		}
		return []int{3}, nil
	}

	h.preview(t, pulid.MustNew("bqi_"), tenant())

	assert.Equal(t, []kernelports.ImagingPageRange{
		{Source: 0, First: 0, Last: imagingkit.ToLastPage, Rotation: 0},
		{Source: 1, First: 0, Last: 0, Rotation: 0},
		{Source: 1, First: 1, Last: 1, Rotation: 90},
	}, ranges)
}

func TestAttachmentRotationsAreKeptByDocument(t *testing.T) {
	pod := pulid.MustNew("doc_")
	bol := pulid.MustNew("doc_")

	got := AttachmentRotations([]*invoice.Attachment{
		{DocumentID: pod, PageRotations: turns(1, 90)},
		{DocumentID: bol},
		nil,
	})

	assert.Equal(t, map[pulid.ID][]billingvocab.PageRotation{pod: turns(1, 90)}, got)
}
