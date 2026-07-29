package templateengine

import (
	"html/template"
	"strings"
)

// A template author writes body content and a stylesheet, not a whole document.
//
// The skeleton is assembled here so the charset, the print colour scheme, and the
// page box are ours to guarantee. An author who forgot a charset would otherwise
// ship an invoice that renders a customer's name as mojibake, and one who omitted
// print-colour-adjust would ship a table whose banding disappears on paper.

// WrapRequest is one rendered body plus the stylesheet that dresses it.
type WrapRequest struct {
	Title    string
	BodyHTML string
	CSS      string
}

// documentSkeleton is intentionally minimal: everything visual belongs to the
// author's stylesheet, and everything here exists because omitting it produces a
// wrong document rather than an ugly one.
var documentSkeleton = template.Must(template.New("document").Parse(
	`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{ .Title }}</title>
<style>
  :root { color-scheme: light; }
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }
  body {
    font-family: -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
    color: #111827;
    background: #ffffff;
    -webkit-print-color-adjust: exact;
    print-color-adjust: exact;
  }
  table { border-collapse: collapse; }
  img { max-width: 100%; }
</style>
{{ if .CSS }}<style>
{{ .CSS }}
</style>{{ end }}
</head>
<body>
{{ .Body }}
</body>
</html>`))

// WrapDocument assembles a printable document from a rendered body and stylesheet.
//
// The body is already-rendered output that has been through the sanitizer, so it is
// inserted as trusted markup. The stylesheet has been through SanitizeCSS. Passing
// unsanitized content here would defeat both.
func WrapDocument(req WrapRequest) string {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Document"
	}

	var b strings.Builder
	b.Grow(len(req.BodyHTML) + len(req.CSS) + 1024)

	// The template's fields are typed as pre-sanitized markup, which is why the
	// escaper leaves them alone. See the doc comment above.
	err := documentSkeleton.Execute(&b, struct {
		Title string
		Body  template.HTML
		CSS   template.CSS
	}{
		Title: title,
		//nolint:gosec // Body is rendered output that already passed SanitizeHTML.
		Body: template.HTML(req.BodyHTML),
		//nolint:gosec // CSS already passed SanitizeCSS.
		CSS: template.CSS(req.CSS),
	})
	if err != nil {
		// The skeleton is a compile-time constant with three string fields; it
		// cannot fail to execute. Returning the body alone still produces something
		// a browser will render rather than an empty page.
		return req.BodyHTML
	}

	return b.String()
}
