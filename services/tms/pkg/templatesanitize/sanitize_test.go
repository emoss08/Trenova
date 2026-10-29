package templatesanitize

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/templatediag"
	"github.com/emoss08/trenova/pkg/templatediagtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSanitizeHTMLRejectsXSSCorpus is the load-bearing security test for
// organization-authored markup. Every entry must be refused at save time, with a
// code the editor can act on.
func TestSanitizeHTMLRejectsXSSCorpus(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantCode string
	}{
		// Script execution.
		{
			"script element",
			`<div><script>alert(1)</script></div>`,
			templatediag.CodeForbiddenElement,
		},
		{
			"script with type",
			`<script type="text/javascript">x()</script>`,
			templatediag.CodeForbiddenElement,
		},
		{"uppercase script", `<SCRIPT>alert(1)</SCRIPT>`, templatediag.CodeForbiddenElement},
		{"mixed case script", `<ScRiPt>alert(1)</ScRiPt>`, templatediag.CodeForbiddenElement},

		// Inline event handlers.
		{
			"img onerror",
			`<img src="data:image/png;base64,AA" onerror="alert(1)">`,
			templatediag.CodeForbiddenAttribute,
		},
		{"body onload", `<body onload="alert(1)">x</body>`, templatediag.CodeForbiddenAttribute},
		{
			"div onmouseover",
			`<div onmouseover="alert(1)">x</div>`,
			templatediag.CodeForbiddenAttribute,
		},
		{
			"uppercase ONERROR",
			`<img src="data:image/png;base64,AA" ONERROR="alert(1)">`,
			templatediag.CodeForbiddenAttribute,
		},
		{
			"onfocus with autofocus",
			`<span onfocus="alert(1)" autofocus>x</span>`,
			templatediag.CodeForbiddenAttribute,
		},

		// Dangerous URL schemes.
		{
			"javascript href",
			`<a href="javascript:alert(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript uppercase",
			`<a href="JAVASCRIPT:alert(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript with spaces",
			`<a href="  javascript:alert(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript entity encoded colon",
			`<a href="javascript&#58;alert(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript percent encoded colon",
			`<a href="javascript%3aalert(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript with embedded newline",
			"<a href=\"java\nscript:alert(1)\">x</a>",
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"javascript with embedded tab",
			"<a href=\"java\tscript:alert(1)\">x</a>",
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"vbscript href",
			`<a href="vbscript:msgbox(1)">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},
		{"file src", `<img src="file:///etc/passwd">`, templatediag.CodeForbiddenURLScheme},
		{"about src", `<img src="about:blank">`, templatediag.CodeForbiddenURLScheme},
		{"blob src", `<img src="blob:http://x/y">`, templatediag.CodeForbiddenURLScheme},
		{
			"view-source",
			`<a href="view-source:http://x">x</a>`,
			templatediag.CodeForbiddenURLScheme,
		},

		// Remote fetches. The renderer has no network route, so these would fail
		// silently at best and probe the internal network at worst.
		{
			"absolute http image",
			`<img src="http://evil.example/x.png">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"absolute https image",
			`<img src="https://evil.example/x.png">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"protocol relative image",
			`<img src="//evil.example/x.png">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"metadata endpoint",
			`<img src="http://169.254.169.254/latest/meta-data/">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"relative path image",
			`<img src="/assets/logo.png">`,
			templatediag.CodeForbiddenURLScheme,
		},

		// data: URIs that are not images.
		{
			"data html",
			`<img src="data:text/html,<script>alert(1)</script>">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"data html base64",
			`<img src="data:text/html;base64,PHNjcmlwdD4=">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"data svg",
			`<img src="data:image/svg+xml;base64,PHN2Zz4=">`,
			templatediag.CodeForbiddenURLScheme,
		},
		{
			"data javascript",
			`<img src="data:text/javascript,alert(1)">`,
			templatediag.CodeForbiddenURLScheme,
		},

		// Framing and embedding.
		{"iframe", `<iframe src="data:text/html,x"></iframe>`, templatediag.CodeForbiddenElement},
		{"object", `<object data="data:x"></object>`, templatediag.CodeForbiddenElement},
		{"embed", `<embed src="data:x">`, templatediag.CodeForbiddenElement},
		{
			"srcdoc attribute",
			`<div srcdoc="<script>x</script>">y</div>`,
			templatediag.CodeForbiddenAttribute,
		},

		// Document-level rewrites.
		{"base element", `<base href="http://evil.example/">`, templatediag.CodeForbiddenElement},
		{
			"link stylesheet",
			`<link rel="stylesheet" href="data:text/css,x">`,
			templatediag.CodeForbiddenElement,
		},
		{
			"meta refresh",
			`<meta http-equiv="refresh" content="0;url=http://evil.example">`,
			templatediag.CodeForbiddenAttribute,
		},

		// SVG and MathML are script vectors.
		{"svg element", `<svg><circle r="1"/></svg>`, templatediag.CodeForbiddenElement},
		{"svg with onload", `<svg onload="alert(1)"></svg>`, templatediag.CodeForbiddenElement},
		{"mathml", `<math><mi>x</mi></math>`, templatediag.CodeForbiddenElement},

		// Forms.
		{
			"form",
			`<form action="data:x"><input name="a"></form>`,
			templatediag.CodeForbiddenElement,
		},
		{"bare input", `<input type="text" name="a">`, templatediag.CodeForbiddenElement},

		// Legacy conditional comments execute in some mail clients.
		{
			"conditional comment",
			`<!--[if IE]><script>alert(1)</script><![endif]-->`,
			templatediag.CodeForbiddenElement,
		},

		// CSS that fetches or executes.
		{
			"style attribute import",
			`<div style="@import url(http://evil.example/x.css)">y</div>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style attribute expression",
			`<div style="width:expression(alert(1))">y</div>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style attribute behavior",
			`<div style="behavior:url(#default#x)">y</div>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style attribute moz binding",
			`<div style="-moz-binding:url(http://evil.example/x)">y</div>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style attribute remote background",
			`<div style="background-image:url(http://evil.example/x.png)">y</div>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style block import",
			`<style>@import "http://evil.example/x.css";</style>`,
			templatediag.CodeForbiddenCSS,
		},
		{
			"style block remote url",
			`<style>body{background:url(https://evil.example/x.png)}</style>`,
			templatediag.CodeForbiddenCSS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := HTML("bodyHtml", tt.source, DocumentOptions())

			require.True(t, diags.HasErrors(),
				"%s must be rejected; got %v", tt.name, templatediagtest.Codes(diags))
			assert.True(t, templatediagtest.HasCode(diags, tt.wantCode),
				"expected code %s, got %v", tt.wantCode, templatediagtest.Codes(diags))
		})
	}
}

// TestSanitizeHTMLAcceptsLegitimateDocumentMarkup guards the other direction: a
// sanitizer that refuses ordinary markup is a sanitizer nobody can author against.
func TestSanitizeHTMLAcceptsLegitimateDocumentMarkup(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"plain text", `Balance due on receipt.`},
		{"headings and paragraphs", `<h1>Invoice</h1><p>Thank you for your business.</p>`},
		{"charge table", `<table><thead><tr><th>Description</th><th>Amount</th></tr></thead>
			<tbody><tr><td>Linehaul</td><td>1,234.50</td></tr></tbody></table>`},
		{"inline styles", `<div style="font-weight:600;color:#111;margin-top:8px">Remit To</div>`},
		{"style block", `<style>body{font-family:Helvetica,Arial,sans-serif}
			.total{font-variant-numeric:tabular-nums}</style>`},
		{
			"data uri image with dimensions",
			`<img src="data:image/png;base64,iVBORw0KGgo=" width="120" height="40" alt="logo">`,
		},
		{"data uri jpeg", `<img src="data:image/jpeg;base64,/9j/4AAQ" alt="logo">`},
		{"mailto link", `<a href="mailto:billing@example.com">Contact billing</a>`},
		{"tel link", `<a href="tel:+15125551234">Call dispatch</a>`},
		{"anchor without href", `<a name="top">Top</a>`},
		{
			"data uri in css",
			`<style>.logo{background-image:url(data:image/png;base64,iVBORw0KGgo=)}</style>`,
		},
		{"css with page rule", `<style>@page{size:letter;margin:20mm}</style>`},
		{
			"semantic markup",
			`<section><article><span class="muted">Terms</span></article></section>`,
		},
		{"template variable in url position", `<img src="{{ .LogoDataURI }}" alt="logo">`},
		{"template variable in css url", `<style>.l{background:url({{ .LogoDataURI }})}</style>`},
		{"empty", ``},
		{"whitespace only", "  \n\t "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := HTML("bodyHtml", tt.source, DocumentOptions())
			assert.False(t, diags.HasErrors(),
				"%s must be accepted; got %v", tt.name, diags.Summary())
		})
	}
}

// TestSanitizeEmailRejectsStyleBlocks documents the one rule that differs by
// channel. Mail clients strip <style>, so an author who relies on it ships a
// document that renders unstyled in a large share of inboxes.
func TestSanitizeEmailRejectsStyleBlocks(t *testing.T) {
	source := `<style>body{color:#111}</style><p>Invoice attached.</p>`

	emailDiags := HTML("bodyHtml", source, EmailOptions())
	require.True(t, emailDiags.HasErrors())
	assert.True(t, templatediagtest.HasCode(emailDiags, templatediag.CodeForbiddenElement))
	assert.Contains(t, emailDiags.Summary(), "stripped by mail clients")

	docDiags := HTML("bodyHtml", source, DocumentOptions())
	assert.False(t, docDiags.HasErrors(), "a PDF may use a real stylesheet")
}

func TestSanitizeHTMLReportsPosition(t *testing.T) {
	source := "<p>line one</p>\n<p>line two</p>\n<script>alert(1)</script>"

	diags := HTML("bodyHtml", source, DocumentOptions())
	require.True(t, diags.HasErrors())

	found := false
	for _, d := range diags {
		if d.Code == templatediag.CodeForbiddenElement {
			assert.Equal(t, 3, d.Line, "the script is on the third line")
			found = true
		}
	}
	assert.True(t, found)
}

func TestSanitizeHTMLReportsFieldForEditorRouting(t *testing.T) {
	diags := HTML("headerHtml", `<script>x</script>`, DocumentOptions())
	require.NotEmpty(t, diags)
	assert.Equal(t, "headerHtml", diags[0].Field)
}

func TestSanitizeCSS(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		{"plain rules", `body{margin:0}h1{font-size:20px}`, false},
		{"data uri background", `.l{background:url(data:image/png;base64,AA)}`, false},
		{"page rule", `@page{size:A4;margin:15mm}`, false},
		{"media query", `@media print{.hide{display:none}}`, false},
		{"font-face with data uri", `@font-face{src:url(data:font/woff2;base64,AA)}`, false},
		{"tabular nums", `.total{font-variant-numeric:tabular-nums}`, false},
		{"empty", ``, false},

		{"import", `@import url(http://evil.example/x.css);`, true},
		{"import quoted", `@import "http://evil.example/x.css";`, true},
		{"import uppercase", `@IMPORT url(http://evil.example/x.css);`, true},
		{"expression", `.a{width:expression(alert(1))}`, true},
		{"expression spaced", `.a{width:expression (alert(1))}`, true},
		{"behavior", `.a{behavior:url(#x)}`, true},
		{"behaviour british", `.a{behaviour:url(#x)}`, true},
		{"moz binding", `.a{-moz-binding:url(http://evil.example/x)}`, true},
		{"remote url", `.a{background:url(https://evil.example/x.png)}`, true},
		{"protocol relative url", `.a{background:url(//evil.example/x.png)}`, true},
		{"relative url", `.a{background:url(/assets/x.png)}`, true},
		{"file url", `.a{background:url(file:///etc/passwd)}`, true},
		{"javascript url", `.a{background:url(javascript:alert(1))}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := CSS("cssContent", tt.source)
			assert.Equal(t, tt.wantErr, diags.HasErrors(),
				"source %q produced %v", tt.source, templatediagtest.Codes(diags))
		})
	}
}

func TestNormalizeURLForSchemeCheck(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "data:image/png;base64,AA", "data:image/png;base64,aa"},
		{"leading whitespace", "   javascript:alert(1)", "javascript:alert(1)"},
		{"entity colon", "javascript&#58;alert(1)", "javascript:alert(1)"},
		{"percent colon", "javascript%3aalert(1)", "javascript:alert(1)"},
		{"percent colon uppercase", "javascript%3Aalert(1)", "javascript:alert(1)"},
		{"embedded newline", "java\nscript:alert(1)", "javascript:alert(1)"},
		{"embedded tab", "java\tscript:alert(1)", "javascript:alert(1)"},
		{"embedded carriage return", "java\rscript:alert(1)", "javascript:alert(1)"},
		{"uppercase", "JAVASCRIPT:ALERT(1)", "javascript:alert(1)"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeURLForSchemeCheck(tt.in))
		})
	}
}

// TestSanitizeHTMLMultipleFindings confirms a single pass reports everything,
// rather than stopping at the first problem and making the author iterate.
func TestSanitizeHTMLMultipleFindings(t *testing.T) {
	source := `<script>a()</script>
		<img src="http://evil.example/x.png" onerror="b()">
		<iframe src="data:text/html,c"></iframe>`

	diags := HTML("bodyHtml", source, DocumentOptions())

	assert.GreaterOrEqual(t, len(diags), 3, "got %v", templatediagtest.Codes(diags))
	assert.True(t, templatediagtest.HasCode(diags, templatediag.CodeForbiddenElement))
	assert.True(t, templatediagtest.HasCode(diags, templatediag.CodeForbiddenAttribute))
}

// TestSanitizeHTMLDoesNotFlagTextContent confirms text is left alone: a document
// that legitimately talks about a <script> tag in prose must still publish.
func TestSanitizeHTMLDoesNotFlagTextContent(t *testing.T) {
	source := `<p>Do not paste &lt;script&gt; tags into the notes field.</p>`

	diags := HTML("bodyHtml", source, DocumentOptions())
	assert.False(t, diags.HasErrors(), strings.Join(templatediagtest.Codes(diags), ","))
}

// TestFetchedAndNavigatedURLsHaveDifferentRules covers a distinction that is easy to
// collapse and expensive to get wrong in either direction.
//
// A src is resolved by the renderer, so it is the one that could reach our network;
// only an inline image is allowed. An href is resolved by the reader in their own
// client much later, so an https link is both safe and necessary — a portal
// invitation with no clickable link has no purpose.
func TestFetchedAndNavigatedURLsHaveDifferentRules(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr bool
	}{
		// Navigation: the reader clicks these.
		{"https href", `<a href="https://app.example.com/invite/abc">Set up</a>`, false},
		{"http href", `<a href="http://app.example.com/x">x</a>`, false},
		{"mailto href", `<a href="mailto:billing@example.com">x</a>`, false},
		{"tel href", `<a href="tel:+15125551234">x</a>`, false},
		{"javascript href is still refused", `<a href="javascript:alert(1)">x</a>`, true},
		{"file href is still refused", `<a href="file:///etc/passwd">x</a>`, true},
		{"relative href has nothing to resolve against", `<a href="/reports">x</a>`, true},

		// Fetching: the renderer resolves these while building the page.
		{"data src", `<img src="data:image/png;base64,iVBORw0KGgo=">`, false},
		{"https src is refused", `<img src="https://cdn.example.com/logo.png">`, true},
		{"http src is refused", `<img src="http://cdn.example.com/logo.png">`, true},
		{"relative src is refused", `<img src="/assets/logo.png">`, true},
		{"metadata endpoint src is refused", `<img src="http://169.254.169.254/">`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := HTML("bodyHtml", tt.source, DocumentOptions())
			assert.Equal(t, tt.wantErr, diags.HasErrors(),
				"source %q produced %v", tt.source, templatediagtest.Codes(diags))
		})
	}
}
