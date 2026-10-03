package docrender

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"strconv"
	"strings"
)

// The parts a Word document needs and no more: the content types, the
// package relationship, the body, and the styles it names.
const (
	contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/></Types>`
	packageRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`
	documentRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`
	stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Georgia" w:hAnsi="Georgia" w:cs="Georgia"/><w:sz w:val="22"/></w:rPr></w:rPrDefault><w:pPrDefault><w:pPr><w:spacing w:after="160" w:line="300" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:rPr><w:sz w:val="44"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:basedOn w:val="Normal"/><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:color w:val="6B7280"/><w:sz w:val="18"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="320" w:after="80"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:rFonts w:ascii="Arial" w:hAnsi="Arial"/><w:b/><w:sz w:val="24"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Heading1"/><w:pPr><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:sz w:val="22"/></w:rPr></w:style><w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/><w:pPr><w:ind w:left="360"/></w:pPr><w:rPr><w:color w:val="4B5563"/></w:rPr></w:style></w:styles>`
)

// DOCX is the document as a Word file.
func DOCX(doc *Document) ([]byte, error) {
	var body strings.Builder
	if doc.Kicker != "" {
		body.WriteString(paragraph("Subtitle", runs(doc.Kicker)))
	}
	body.WriteString(paragraph("Title", run(doc.Title, "")))
	if doc.Byline != "" {
		body.WriteString(paragraph("Subtitle", run(doc.Byline, "")))
	}
	for _, block := range Parse(doc.Body) {
		writeDOCXBlock(&body, block)
	}
	if len(doc.Sources) > 0 {
		body.WriteString(paragraph("Heading2", run("Sources", "")))
		for _, source := range doc.Sources {
			line := "[" + strconv.Itoa(source.N) + "] " + source.Label
			if source.Tool != "" {
				line += " · " + source.Tool
			}
			if source.Detail != "" {
				line += " · " + source.Detail
			}
			body.WriteString(paragraph("", run(line, "")))
		}
	}

	document := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		body.String() +
		`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr>` +
		`</w:body></w:document>`

	var out bytes.Buffer
	archive := zip.NewWriter(&out)
	for _, part := range []struct{ name, content string }{
		{"[Content_Types].xml", contentTypesXML},
		{"_rels/.rels", packageRelsXML},
		{"word/_rels/document.xml.rels", documentRelsXML},
		{"word/styles.xml", stylesXML},
		{"word/document.xml", document},
	} {
		writer, err := archive.Create(part.name)
		if err != nil {
			return nil, err
		}
		if _, err = writer.Write([]byte(part.content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func writeDOCXBlock(b *strings.Builder, block Block) {
	switch block.Kind {
	case BlockHeading:
		style := "Heading1"
		if block.Level > 2 {
			style = "Heading2"
		}
		b.WriteString(paragraph(style, runs(block.Text)))
	case BlockList:
		for idx, item := range block.Items {
			marker := "•\t"
			if block.Ordered {
				marker = strconv.Itoa(idx+1) + ".\t"
			}
			b.WriteString(`<w:p><w:pPr><w:ind w:left="360" w:hanging="360"/></w:pPr>` +
				run(marker, "") + runs(item) + `</w:p>`)
		}
	case BlockTable:
		b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:tblBorders>` +
			`<w:insideH w:val="single" w:sz="4" w:color="E3E5E9"/>` +
			`<w:bottom w:val="single" w:sz="4" w:color="E3E5E9"/></w:tblBorders></w:tblPr>`)
		b.WriteString(tableRow(block.Header, true))
		for _, row := range block.Rows {
			b.WriteString(tableRow(row, false))
		}
		b.WriteString(`</w:tbl>`)
		b.WriteString(paragraph("", ""))
	case BlockQuote:
		b.WriteString(paragraph("Quote", runs(block.Text)))
	case BlockRule:
		b.WriteString(`<w:p><w:pPr><w:pBdr><w:bottom w:val="single" w:sz="4" w:color="E3E5E9"/></w:pBdr></w:pPr></w:p>`)
	case BlockParagraph:
		b.WriteString(paragraph("", runs(block.Text)))
	}
}

func tableRow(cells []string, header bool) string {
	var b strings.Builder
	b.WriteString(`<w:tr>`)
	for _, cell := range cells {
		content := runs(cell)
		if header {
			content = run(PlainText(cell), "<w:b/>")
		}
		b.WriteString(`<w:tc><w:p>` + content + `</w:p></w:tc>`)
	}
	b.WriteString(`</w:tr>`)

	return b.String()
}

func paragraph(style, content string) string {
	if style == "" {
		return `<w:p>` + content + `</w:p>`
	}

	return `<w:p><w:pPr><w:pStyle w:val="` + style + `"/></w:pPr>` + content + `</w:p>`
}

func runs(text string) string {
	var b strings.Builder
	for _, span := range Inline(text) {
		switch span.Kind {
		case SpanBold:
			b.WriteString(run(span.Text, "<w:b/>"))
		case SpanItalic:
			b.WriteString(run(span.Text, "<w:i/>"))
		case SpanCode:
			b.WriteString(run(span.Text, `<w:rFonts w:ascii="Consolas" w:hAnsi="Consolas"/>`))
		case SpanCite:
			b.WriteString(run(span.Text, `<w:vertAlign w:val="superscript"/><w:color w:val="4A5FC1"/>`))
		case SpanLink, SpanText:
			b.WriteString(run(span.Text, ""))
		}
	}

	return b.String()
}

func run(text, properties string) string {
	var escaped bytes.Buffer
	_ = xml.EscapeText(&escaped, []byte(text))
	out := `<w:r>`
	if properties != "" {
		out += `<w:rPr>` + properties + `</w:rPr>`
	}

	return out + `<w:t xml:space="preserve">` + escaped.String() + `</w:t></w:r>`
}
