package officedoc_test

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/officedoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func zipFiles(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.Black)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

const (
	wNS   = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`
	rNS   = `xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"`
	aNS   = `xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"`
	pNS   = `xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"`
	mcNS  = `xmlns:mc="http://schemas.openxmlformats.org/markup-compatibility/2006"`
	relNS = `xmlns="http://schemas.openxmlformats.org/package/2006/relationships"`
)

const rootRels = `<?xml version="1.0"?><Relationships ` + relNS + `>` +
	`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="%s"/>` +
	`</Relationships>`

func docxPackage(t *testing.T, body string) []byte {
	t.Helper()
	return zipFiles(t, map[string][]byte{
		"_rels/.rels": []byte(strings.Replace(rootRels, "%s", "word/document.xml", 1)),
		"word/document.xml": []byte(`<?xml version="1.0"?><w:document ` + wNS + ` ` + rNS + ` ` + aNS + ` ` + mcNS + `><w:body>` +
			body + `</w:body></w:document>`),
		"word/_rels/document.xml.rels": []byte(`<?xml version="1.0"?><Relationships ` + relNS + `>` +
			`<Relationship Id="rIdImg" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/scan%20one.png"/>` +
			`<Relationship Id="rIdExt" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="https://example.com/x.png" TargetMode="External"/>` +
			`</Relationships>`),
		"word/media/scan one.png": pngBytes(t),
	})
}

func read(t *testing.T, data []byte, format services.OfficeFormat) []services.OfficePage {
	t.Helper()
	pages, err := officedoc.New().Read(t.Context(), data, format)
	require.NoError(t, err)
	return pages
}

func TestReadDOCXSplitsPagesAndKeepsStructure(t *testing.T) {
	t.Parallel()

	body := `<w:p><w:pPr><w:tabs><w:tab w:val="left" w:pos="720"/></w:tabs></w:pPr>` +
		`<w:r><w:t>Rate</w:t><w:tab/><w:t xml:space="preserve">$1,359.56</w:t></w:r></w:p>` +
		`<w:p><w:r><w:t>PO</w:t><w:br/><w:t>77812</w:t></w:r>` +
		`<w:r><w:instrText>PAGE \* MERGEFORMAT</w:instrText></w:r></w:p>` +
		`<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Origin</w:t></w:r></w:p></w:tc>` +
		`<w:tc><w:p><w:r><w:t>Destination</w:t></w:r></w:p></w:tc></w:tr></w:tbl>` +
		`<w:p><w:r><w:br w:type="page"/></w:r></w:p>` +
		`<w:p><w:r><w:lastRenderedPageBreak/><w:t>Second page</w:t></w:r></w:p>` +
		`<w:p><w:r><mc:AlternateContent><mc:Choice Requires="wps"><w:drawing><a:blip r:embed="rIdImg"/></w:drawing></mc:Choice>` +
		`<mc:Fallback><w:pict><w:t>Fallback copy</w:t></w:pict></mc:Fallback></mc:AlternateContent></w:r></w:p>` +
		`<w:p><w:r><w:drawing><a:blip r:embed="rIdExt"/></w:drawing></w:r></w:p>` +
		`<w:p><w:pPr><w:pageBreakBefore/></w:pPr><w:r><w:t>Third page</w:t></w:r></w:p>` +
		`<w:p><w:r><w:del><w:delText>gone</w:delText></w:del></w:r></w:p>`

	pages := read(t, docxPackage(t, body), services.OfficeFormatDOCX)

	require.Len(t, pages, 3)
	assert.Equal(t, "Rate\t$1,359.56\nPO\n77812\nOrigin\tDestination", pages[0].Text)
	assert.Empty(t, pages[0].Images)

	assert.Equal(t, "Second page", pages[1].Text)
	require.Len(t, pages[1].Images, 1)
	assert.Equal(t, ".png", pages[1].Images[0].Ext)
	assert.Equal(t, "scan one.png", pages[1].Images[0].Name)
	assert.Equal(t, pngBytes(t), pages[1].Images[0].Data)

	assert.Equal(t, "Third page", pages[2].Text)
}

func TestReadPPTXFollowsSlideOrder(t *testing.T) {
	t.Parallel()

	slide := func(text string, withImage bool) []byte {
		pic := ""
		if withImage {
			pic = `<p:pic><p:blipFill><a:blip r:embed="rId2"/></p:blipFill></p:pic>`
		}
		return []byte(`<?xml version="1.0"?><p:sld ` + pNS + ` ` + aNS + ` ` + rNS + `><p:cSld><p:spTree>` +
			`<p:sp><p:txBody><a:p><a:r><a:t>` + text + `</a:t></a:r><a:br/><a:r><a:t>line two</a:t></a:r></a:p></p:txBody></p:sp>` +
			pic + `</p:spTree></p:cSld></p:sld>`)
	}
	slideRels := []byte(`<?xml version="1.0"?><Relationships ` + relNS + `>` +
		`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/>` +
		`</Relationships>`)

	data := zipFiles(t, map[string][]byte{
		"_rels/.rels": []byte(strings.Replace(rootRels, "%s", "ppt/presentation.xml", 1)),
		"ppt/presentation.xml": []byte(`<?xml version="1.0"?><p:presentation ` + pNS + ` ` + rNS + `><p:sldIdLst>` +
			`<p:sldId id="256" r:id="rId8"/><p:sldId id="257" r:id="rId7"/></p:sldIdLst></p:presentation>`),
		"ppt/_rels/presentation.xml.rels": []byte(`<?xml version="1.0"?><Relationships ` + relNS + `>` +
			`<Relationship Id="rId7" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>` +
			`<Relationship Id="rId8" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="/ppt/slides/slide2.xml"/>` +
			`</Relationships>`),
		"ppt/slides/slide1.xml":            slide("Second slide", false),
		"ppt/slides/slide2.xml":            slide("First slide", true),
		"ppt/slides/_rels/slide2.xml.rels": slideRels,
		"ppt/media/image1.png":             pngBytes(t),
	})

	pages := read(t, data, services.OfficeFormatPPTX)

	require.Len(t, pages, 2)
	assert.Equal(t, "First slide\nline two", pages[0].Text)
	require.Len(t, pages[0].Images, 1)
	assert.Equal(t, "Second slide\nline two", pages[1].Text)
	assert.Empty(t, pages[1].Images)
}

func TestReadXLSXReadsEverySheet(t *testing.T) {
	t.Parallel()

	book := excelize.NewFile()
	require.NoError(t, book.SetSheetRow("Sheet1", "A1", &[]any{"Lane", "Rate", ""}))
	require.NoError(t, book.SetSheetRow("Sheet1", "A3", &[]any{"ATL-DAL", 1359.56}))
	_, err := book.NewSheet("Fuel")
	require.NoError(t, err)
	require.NoError(t, book.SetCellValue("Fuel", "B2", "Surcharge"))
	require.NoError(t, book.AddPictureFromBytes("Fuel", "D4", &excelize.Picture{
		Extension: ".png",
		File:      pngBytes(t),
		Format:    &excelize.GraphicOptions{},
	}))
	var buf bytes.Buffer
	require.NoError(t, book.Write(&buf))
	require.NoError(t, book.Close())

	pages := read(t, buf.Bytes(), services.OfficeFormatXLSX)

	require.Len(t, pages, 2)
	assert.Equal(t, "Lane\tRate\nATL-DAL\t1359.56", pages[0].Text)
	assert.Equal(t, "Surcharge", pages[1].Text)
	require.Len(t, pages[1].Images, 1)
	assert.Equal(t, ".png", pages[1].Images[0].Ext)
}

func TestReadEPUBReadsTheSpine(t *testing.T) {
	t.Parallel()

	chapter := func(body string) []byte {
		return []byte(`<?xml version="1.0"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>Ignored</title>` +
			`<style>p { color: red }</style></head><body>` + body + `</body></html>`)
	}

	data := zipFiles(t, map[string][]byte{
		"mimetype": []byte("application/epub+zip"),
		"META-INF/container.xml": []byte(`<?xml version="1.0"?><container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">` +
			`<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`),
		"OEBPS/content.opf": []byte(`<?xml version="1.0"?><package xmlns="http://www.idpf.org/2007/opf"><manifest>` +
			`<item id="c2" href="text/two.xhtml" media-type="application/xhtml+xml"/>` +
			`<item id="c1" href="text/one.xhtml" media-type="application/xhtml+xml"/>` +
			`</manifest><spine><itemref idref="c1"/><itemref idref="c2"/><itemref idref="missing"/></spine></package>`),
		"OEBPS/text/one.xhtml": chapter(`<h1>Bill of   lading</h1><p>Shipper: <b>Acme</b>
			Freight &amp; Co</p><script>alert(1)</script><table><tr><td>Pieces</td><td>12</td></tr></table>`),
		"OEBPS/text/two.xhtml": chapter(`<p>Signed</p><img src="../images/sig.png"/><img src="https://example.com/x.png"/>`),
		"OEBPS/images/sig.png": pngBytes(t),
	})

	pages := read(t, data, services.OfficeFormatEPUB)

	require.Len(t, pages, 2)
	assert.Equal(t, "Bill of lading\nShipper: Acme Freight & Co\nPieces\t12", pages[0].Text)
	assert.Equal(t, "Signed", pages[1].Text)
	require.Len(t, pages[1].Images, 1)
	assert.Equal(t, "sig.png", pages[1].Images[0].Name)
}

func TestReadRejectsUnreadableInput(t *testing.T) {
	t.Parallel()

	reader := officedoc.New()

	for _, format := range []services.OfficeFormat{
		services.OfficeFormatDOCX,
		services.OfficeFormatXLSX,
		services.OfficeFormatPPTX,
		services.OfficeFormatEPUB,
	} {
		_, err := reader.Read(t.Context(), []byte("not a zip"), format)
		require.ErrorIs(t, err, services.ErrOfficeDocumentUnreadable, format)
	}

	_, err := reader.Read(t.Context(), zipFiles(t, map[string][]byte{"x": nil}), services.OfficeFormatDOCX)
	require.ErrorIs(t, err, services.ErrOfficeDocumentUnreadable)

	_, err = reader.Read(t.Context(), zipFiles(t, map[string][]byte{"x": nil}), "odt")
	require.ErrorIs(t, err, services.ErrOfficeDocumentUnreadable)
}

func TestReadRefusesOversizedParts(t *testing.T) {
	t.Parallel()

	huge := `<w:p><w:r><w:t>` + strings.Repeat("a", 65<<20) + `</w:t></w:r></w:p>`
	_, err := officedoc.New().Read(t.Context(), docxPackage(t, huge), services.OfficeFormatDOCX)
	require.ErrorIs(t, err, services.ErrOfficeDocumentUnreadable)
}

func TestReadStopsWhenCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	body := strings.Repeat(`<w:p><w:r><w:t>row</w:t></w:r></w:p>`, 2000)
	_, err := officedoc.New().Read(ctx, docxPackage(t, body), services.OfficeFormatDOCX)
	require.ErrorIs(t, err, context.Canceled)
}

func TestOfficeFormatOf(t *testing.T) {
	t.Parallel()

	format, ok := services.OfficeFormatOf(
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document; charset=binary", "x.bin",
	)
	assert.True(t, ok)
	assert.Equal(t, services.OfficeFormatDOCX, format)

	format, ok = services.OfficeFormatOf("application/octet-stream", "Rates.XLSX")
	assert.True(t, ok)
	assert.Equal(t, services.OfficeFormatXLSX, format)

	_, ok = services.OfficeFormatOf("application/pdf", "x.pdf")
	assert.False(t, ok)
	_, ok = services.OfficeFormatOf("application/msword", "x.doc")
	assert.False(t, ok)
}
