package documentintelligencejobs

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/officedoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePDFPage struct {
	text   string
	layout *services.PDFPageLayout
}

type fakePDFDocument struct {
	pages  []fakePDFPage
	closed bool
}

func (d *fakePDFDocument) PageCount() int { return len(d.pages) }

func (d *fakePDFDocument) RenderPage(context.Context, int, int) (*image.RGBA, error) {
	return image.NewRGBA(image.Rect(0, 0, 2, 2)), nil
}

func (d *fakePDFDocument) PageText(_ context.Context, page int) (string, error) {
	return d.pages[page].text, nil
}

func (d *fakePDFDocument) PageLayout(_ context.Context, page int) (*services.PDFPageLayout, error) {
	if d.pages[page].layout == nil {
		return nil, errors.New("no layout")
	}
	return d.pages[page].layout, nil
}

func (d *fakePDFDocument) Close() error {
	d.closed = true
	return nil
}

type fakePDFReader struct {
	doc *fakePDFDocument
	err error
}

func (r *fakePDFReader) Open(context.Context, []byte) (services.PDFDocument, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.doc, nil
}

func extractionActivities(pdf services.PDFReader) *Activities {
	return &Activities{
		logger:       zap.NewNop(),
		cfg:          &config.AIConfig{},
		pdfReader:    pdf,
		officeReader: officedoc.New(),
	}
}

func TestExtractContentReadsNativePDFPages(t *testing.T) {
	t.Parallel()

	pdfDoc := &fakePDFDocument{pages: []fakePDFPage{
		{
			text: "Rate: $1,359.56",
			layout: &services.PDFPageLayout{
				Width: 600, Height: 800,
				Lines: []services.PDFTextLine{{Text: "Rate: $1,359.56", Left: 60, Top: 80, Width: 75, Height: 12}},
			},
		},
		{text: "Page two"},
		{text: "  "},
	}}
	activities := extractionActivities(&fakePDFReader{doc: pdfDoc})

	result, err := activities.extractContent(
		t.Context(),
		&document.Document{FileType: "application/pdf", OriginalName: "rate.pdf"},
		[]byte("%PDF"),
		&tenant.DocumentControl{EnableOCR: false},
	)
	require.NoError(t, err)

	assert.True(t, pdfDoc.closed)
	assert.Equal(t, 3, result.PageCount)
	require.Len(t, result.Pages, 3)
	assert.Equal(t, "Rate: $1,359.56", result.Pages[0].Text)
	lines, ok := result.Pages[0].Metadata[documentcontent.MetadataLines].([]documentcontent.LayoutLine)
	require.True(t, ok)
	require.Len(t, lines, 1)
	assert.InDelta(t, 0.1, lines[0].X, 0.0001)
	assert.Equal(t, "Page two", result.Pages[1].Text)
	assert.NotContains(t, result.Pages[1].Metadata, documentcontent.MetadataLines)
	assert.Equal(t, "native_skipped", result.Pages[2].Metadata["extractionMode"])
}

func TestExtractContentFailsOnUnreadablePDF(t *testing.T) {
	t.Parallel()

	activities := extractionActivities(&fakePDFReader{err: services.ErrPDFUnreadable})

	_, err := activities.extractContent(
		t.Context(),
		&document.Document{FileType: "application/octet-stream", OriginalName: "scan.PDF"},
		[]byte("junk"),
		&tenant.DocumentControl{},
	)
	require.ErrorIs(t, err, services.ErrPDFUnreadable)
}

func TestExtractContentReadsOfficeDocuments(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	require.NoError(t, err)
	_, err = w.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		`<w:p><w:r><w:t>Bill of lading 4471</w:t></w:r></w:p>` +
		`<w:p><w:r><w:br w:type="page"/></w:r></w:p>` +
		`<w:p><w:r><w:t>Consignee</w:t></w:r></w:p></w:body></w:document>`))
	require.NoError(t, err)
	require.NoError(t, zw.Close())

	activities := extractionActivities(nil)
	result, err := activities.extractContent(
		t.Context(),
		&document.Document{
			FileType:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			OriginalName: "bol",
		},
		buf.Bytes(),
		&tenant.DocumentControl{EnableOCR: true},
	)
	require.NoError(t, err)

	assert.Equal(t, 2, result.PageCount)
	require.Len(t, result.Pages, 2)
	assert.Equal(t, "Bill of lading 4471", result.Pages[0].Text)
	assert.Equal(t, documentcontent.SourceKindNative, result.Pages[0].SourceKind)
	assert.Equal(t, "Consignee", result.Pages[1].Text)
	assert.Contains(t, result.Text, "Bill of lading 4471")
}

func TestExtractContentSkipsOCRForImageOnlyOfficePagesWhenDisabled(t *testing.T) {
	t.Parallel()

	activities := extractionActivities(nil)
	activities.officeReader = officeReaderFunc(func() []services.OfficePage {
		return []services.OfficePage{{Images: []services.OfficeImage{{Name: "a.png", Ext: ".png", Data: []byte{1}}}}}
	})

	result, err := activities.extractContent(
		t.Context(),
		&document.Document{FileType: "application/octet-stream", OriginalName: "scan.docx"},
		[]byte("zip"),
		&tenant.DocumentControl{EnableOCR: false},
	)
	require.NoError(t, err)
	require.Len(t, result.Pages, 1)
	assert.Equal(t, "native_skipped", result.Pages[0].Metadata["extractionMode"])
}

type officeReaderFunc func() []services.OfficePage

func (f officeReaderFunc) Read(context.Context, []byte, services.OfficeFormat) ([]services.OfficePage, error) {
	return f(), nil
}
