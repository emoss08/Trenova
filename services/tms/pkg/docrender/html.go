package docrender

import (
	"html"
	"strconv"
	"strings"
)

// Document is a write-up and what is printed around it.
type Document struct {
	Title   string
	Kicker  string
	Byline  string
	Body    string
	Sources []Source
}

// Source is what a citation mark points to, listed at the end.
type Source struct {
	N      int
	Label  string
	Tool   string
	Detail string
}

const pageStyle = `body{margin:0;font:400 11.5pt/1.6 'Newsreader',Georgia,'Times New Roman',serif;color:#1f2329}
.kick{font:500 8pt 'IBM Plex Mono',Menlo,monospace;letter-spacing:.06em;text-transform:uppercase;color:#4a5fc1}
h1{margin:8pt 0 6pt;font:500 22pt/1.2 'Newsreader',Georgia,serif;color:#111}
.by{padding-bottom:12pt;margin-bottom:14pt;border-bottom:1px solid #e3e5e9;font:9pt Helvetica,Arial,sans-serif;color:#6b7280}
h2,h3,h4{margin:18pt 0 4pt;font:600 11pt Helvetica,Arial,sans-serif;color:#111}
p{margin:0 0 9pt}ul,ol{margin:0 0 10pt;padding-left:18pt}li{margin:2pt 0}
blockquote{margin:0 0 10pt;padding-left:10pt;border-left:2px solid #d1d5db;color:#4b5563}
table{width:100%;border-collapse:collapse;margin:4pt 0 10pt;font:9pt Helvetica,Arial,sans-serif}
th{text-align:left;font-weight:500;color:#6b7280;border-bottom:1px solid #e3e5e9;padding:0 6pt 5pt 0}
td{border-bottom:1px solid #eef0f3;padding:5pt 6pt 5pt 0}
code{font:9pt 'IBM Plex Mono',Menlo,monospace}
sup{font:500 7pt 'IBM Plex Mono',Menlo,monospace;color:#4a5fc1}
.src{margin-top:24pt;padding-top:10pt;border-top:1px solid #e3e5e9;font:9pt Helvetica,Arial,sans-serif;color:#374151}
.src h4{margin:0 0 6pt;font:500 8pt 'IBM Plex Mono',Menlo,monospace;letter-spacing:.06em;text-transform:uppercase;color:#6b7280}
.src div{margin:3pt 0}.src code{color:#6b7280}`

// HTML is the document as a printable page.
func HTML(doc *Document) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>`)
	b.WriteString(html.EscapeString(doc.Title))
	b.WriteString(`</title><style>`)
	b.WriteString(pageStyle)
	b.WriteString(`</style></head><body>`)
	if doc.Kicker != "" {
		b.WriteString(`<div class="kick">` + html.EscapeString(doc.Kicker) + `</div>`)
	}
	b.WriteString(`<h1>` + html.EscapeString(doc.Title) + `</h1>`)
	if doc.Byline != "" {
		b.WriteString(`<div class="by">` + html.EscapeString(doc.Byline) + `</div>`)
	}
	for _, block := range Parse(doc.Body) {
		writeHTMLBlock(&b, block)
	}
	if len(doc.Sources) > 0 {
		b.WriteString(`<section class="src"><h4>Sources</h4>`)
		for _, source := range doc.Sources {
			b.WriteString(`<div><sup>` + strconv.Itoa(source.N) + `</sup> ` + html.EscapeString(source.Label))
			if source.Tool != "" {
				b.WriteString(` <code>` + html.EscapeString(source.Tool) + `</code>`)
			}
			if source.Detail != "" {
				b.WriteString(` · ` + html.EscapeString(source.Detail))
			}
			b.WriteString(`</div>`)
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`</body></html>`)

	return b.String()
}

func writeHTMLBlock(b *strings.Builder, block Block) {
	switch block.Kind {
	case BlockHeading:
		level := strconv.Itoa(min(block.Level+1, 4))
		b.WriteString("<h" + level + ">" + inlineHTML(block.Text) + "</h" + level + ">")
	case BlockList:
		tag := "ul"
		if block.Ordered {
			tag = "ol"
		}
		b.WriteString("<" + tag + ">")
		for _, item := range block.Items {
			b.WriteString("<li>" + inlineHTML(item) + "</li>")
		}
		b.WriteString("</" + tag + ">")
	case BlockTable:
		b.WriteString("<table><thead><tr>")
		for _, cell := range block.Header {
			b.WriteString("<th>" + inlineHTML(cell) + "</th>")
		}
		b.WriteString("</tr></thead><tbody>")
		for _, row := range block.Rows {
			b.WriteString("<tr>")
			for _, cell := range row {
				b.WriteString("<td>" + inlineHTML(cell) + "</td>")
			}
			b.WriteString("</tr>")
		}
		b.WriteString("</tbody></table>")
	case BlockQuote:
		b.WriteString("<blockquote>" + inlineHTML(block.Text) + "</blockquote>")
	case BlockRule:
		b.WriteString("<hr>")
	case BlockParagraph:
		b.WriteString("<p>" + inlineHTML(block.Text) + "</p>")
	}
}

func inlineHTML(text string) string {
	var b strings.Builder
	for _, span := range Inline(text) {
		escaped := html.EscapeString(span.Text)
		switch span.Kind {
		case SpanBold:
			b.WriteString("<b>" + escaped + "</b>")
		case SpanItalic:
			b.WriteString("<i>" + escaped + "</i>")
		case SpanCode:
			b.WriteString("<code>" + escaped + "</code>")
		case SpanLink:
			b.WriteString(escaped)
		case SpanCite:
			b.WriteString("<sup>" + escaped + "</sup>")
		case SpanText:
			b.WriteString(escaped)
		}
	}

	return b.String()
}
