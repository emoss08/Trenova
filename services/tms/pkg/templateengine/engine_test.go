package templateengine

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	engine, err := New(Config{})
	require.NoError(t, err)
	return engine
}

// invoiceSample mirrors the shape a real invoice context has: scalars, nested
// objects, and a collection whose elements have their own fields.
type invoiceSample struct {
	InvoiceNumber string
	Total         decimal.Decimal
	CurrencyCode  string
	InvoiceDate   int64
	Timezone      string
	Memo          string
	BillTo        addressSample
	RemitTo       addressSample
	ChargeRows    []chargeSample
	LogoDataURI   string
}

type addressSample struct {
	Name  string
	Lines []string
}

type chargeSample struct {
	Description string
	Quantity    string
	Amount      string
}

func newInvoiceSample() invoiceSample {
	return invoiceSample{
		InvoiceNumber: "INV-1024",
		Total:         decimal.RequireFromString("1234.50"),
		CurrencyCode:  "USD",
		InvoiceDate:   1784131200,
		Timezone:      "America/Chicago",
		Memo:          "Thanks for your business",
		BillTo:        addressSample{Name: "ACME, Inc.", Lines: []string{"1 Main St", "Austin, TX"}},
		RemitTo:       addressSample{Name: "Trenova", Lines: []string{"PO Box 1"}},
		ChargeRows: []chargeSample{
			{Description: "Linehaul", Quantity: "1", Amount: "1200.00"},
			{Description: "Fuel", Quantity: "1", Amount: "34.50"},
		},
		LogoDataURI: "data:image/png;base64,iVBORw0KGgo=",
	}
}

func invoicePaths() []string {
	return []string{
		"InvoiceNumber", "Total", "CurrencyCode", "InvoiceDate", "Timezone", "Memo",
		"BillTo", "RemitTo", "ChargeRows", "LogoDataURI",
	}
}

func TestParseAcceptsAValidDocumentTemplate(t *testing.T) {
	engine := testEngine(t)

	source := `<h1>Invoice {{ .InvoiceNumber }}</h1>
		<div>{{ .BillTo.Name }}</div>
		{{ range .BillTo.Lines }}<div>{{ . }}</div>{{ end }}
		<table>{{ range .ChargeRows }}<tr><td>{{ .Description }}</td><td>{{ .Amount }}</td></tr>{{ end }}</table>
		<div class="total">{{ money .CurrencyCode .Total }}</div>
		<div>{{ date .InvoiceDate .Timezone }}</div>
		<p>{{ .Memo | default "No memo" }}</p>`

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Kind:         "invoice.pdf",
		Channel:      ChannelPDF,
		Source:       source,
		AllowedPaths: invoicePaths(),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())
	require.NotNil(t, compiled)

	out, err := engine.Render(t.Context(), compiled, newInvoiceSample())
	require.NoError(t, err)
	assert.Contains(t, out, "INV-1024")
	assert.Contains(t, out, "USD 1234.50")
	assert.Contains(t, out, "Linehaul")
	assert.Contains(t, out, "2026-07-15")
}

func TestParseReportsSyntaxErrorWithPosition(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:   "bodyHtml",
		Kind:    "invoice.pdf",
		Channel: ChannelPDF,
		Source:  "<p>line one</p>\n<p>{{ .Broken </p>",
	})
	require.True(t, diags.HasErrors())
	assert.Nil(t, compiled, "a failed parse must never return something renderable")
	assert.True(t, hasCode(diags, CodeSyntaxError))
	assert.Positive(t, diags[0].Line, "the editor needs a line to jump to")
}

// TestUnknownVariableSeverityDependsOnStrict is the drafting-versus-publishing
// asymmetry: a half-typed variable must not block a save, but it must block a
// publish, because it renders as blank space on a customer's invoice.
func TestUnknownVariableSeverityDependsOnStrict(t *testing.T) {
	engine := testEngine(t)
	source := `<p>{{ .Totl }}</p>`

	draft, draftDiags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       source,
		AllowedPaths: invoicePaths(),
		Strict:       false,
	})
	assert.False(t, draftDiags.HasErrors(), "drafting tolerates an unknown path")
	assert.NotNil(t, draft)
	require.NotEmpty(t, draftDiags)
	assert.Equal(t, SeverityWarning, draftDiags[0].Severity)
	assert.Contains(t, draftDiags[0].Message, `Did you mean "Total"?`)

	_, publishDiags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       source,
		AllowedPaths: invoicePaths(),
		Strict:       true,
	})
	assert.True(t, publishDiags.HasErrors(), "publishing rejects an unknown path")
	assert.True(t, hasCode(publishDiags, CodeUnknownVariable))
}

func TestNestedPathsResolveThroughRangeAndWith(t *testing.T) {
	engine := testEngine(t)

	tests := []struct {
		name      string
		source    string
		wantPaths []string
	}{
		{
			"range binds dot to the element",
			`{{ range .ChargeRows }}{{ .Description }}{{ end }}`,
			[]string{"ChargeRows", "ChargeRows.Description"},
		},
		{
			"range with an index variable",
			`{{ range $i, $row := .ChargeRows }}{{ $row.Amount }}{{ end }}`,
			[]string{"ChargeRows", "ChargeRows.Amount"},
		},
		{
			"with binds dot",
			`{{ with .BillTo }}{{ .Name }}{{ end }}`,
			[]string{"BillTo", "BillTo.Name"},
		},
		{
			"nested range",
			`{{ range .ChargeRows }}{{ range .Description }}{{ . }}{{ end }}{{ end }}`,
			[]string{"ChargeRows", "ChargeRows.Description"},
		},
		{
			"root escape via dollar",
			`{{ range .ChargeRows }}{{ $.InvoiceNumber }}{{ end }}`,
			[]string{"ChargeRows", "InvoiceNumber"},
		},
		{
			"else branch resolves against the outer scope",
			`{{ with .BillTo }}{{ .Name }}{{ else }}{{ .InvoiceNumber }}{{ end }}`,
			[]string{"BillTo", "BillTo.Name", "InvoiceNumber"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, diags := engine.Parse(&ParseRequest{
				Field:        "bodyHtml",
				Channel:      ChannelPDF,
				Source:       tt.source,
				AllowedPaths: invoicePaths(),
				Strict:       true,
			})
			require.False(t, diags.HasErrors(), diags.Summary())
			for _, want := range tt.wantPaths {
				assert.Contains(t, compiled.Paths, want)
			}
		})
	}
}

// TestRequiredPathsBlockPublish protects the fields that make a document a legal
// artifact. An invoice without a total is one a customer can refuse to pay.
func TestRequiredPathsBlockPublish(t *testing.T) {
	engine := testEngine(t)
	required := []string{"InvoiceNumber", "Total", "BillTo", "RemitTo"}

	// The check runs over the union of paths across a version's channels, because a
	// version is published as a whole: a subject line cannot restate a charge table.
	pathsFor := func(t *testing.T, source string) []string {
		t.Helper()
		compiled, diags := engine.Parse(&ParseRequest{
			Field:        "bodyHtml",
			Channel:      ChannelPDF,
			Source:       source,
			AllowedPaths: invoicePaths(),
		})
		require.False(t, diags.HasErrors(), diags.Summary())
		return compiled.Paths
	}

	t.Run("dropping a required field is rejected", func(t *testing.T) {
		paths := pathsFor(t, `<p>{{ .InvoiceNumber }} {{ .BillTo.Name }} {{ .RemitTo.Name }}</p>`)

		diags := CheckRequiredPaths("bodyHtml", paths, required)
		require.True(t, diags.HasErrors())
		assert.True(t, hasCode(diags, CodeRequiredVariableMissing))
		assert.Contains(t, diags.Summary(), `"Total"`)
	})

	t.Run("referencing a child satisfies the parent", func(t *testing.T) {
		paths := pathsFor(t, `<p>{{ .InvoiceNumber }} {{ .Total }}</p>
			<div>{{ .BillTo.Name }}</div><div>{{ .RemitTo.Name }}</div>`)

		diags := CheckRequiredPaths("bodyHtml", paths, required)
		assert.False(t, diags.HasErrors(), diags.Summary())
	})

	t.Run("a field used only in a condition still counts as referenced", func(t *testing.T) {
		// {{ if .Total }} reads Total whether or not the body prints it. Missing
		// this made every conditionally-rendered section look unreferenced.
		paths := pathsFor(t, `{{ if .Total }}<p>{{ .InvoiceNumber }}</p>{{ end }}
			<div>{{ .BillTo.Name }}</div><div>{{ .RemitTo.Name }}</div>`)

		assert.Contains(t, paths, "Total")
		diags := CheckRequiredPaths("bodyHtml", paths, required)
		assert.False(t, diags.HasErrors(), diags.Summary())
	})

	t.Run("no requirement is no error", func(t *testing.T) {
		diags := CheckRequiredPaths("bodyHtml", pathsFor(t, `<p>x</p>`), nil)
		assert.False(t, diags.HasErrors())
	})
}

// TestConditionsAreAnalyzed covers the analyzer bug directly: a variable read only
// by an if or a with condition must still be recorded.
func TestConditionsAreAnalyzed(t *testing.T) {
	engine := testEngine(t)

	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"if condition", `{{ if .Memo }}x{{ end }}`, "Memo"},
		{"if with a function", `{{ if eq .CurrencyCode "USD" }}x{{ end }}`, "CurrencyCode"},
		{"else if condition", `{{ if .Memo }}a{{ else if .Timezone }}b{{ end }}`, "Timezone"},
		{"with condition", `{{ with .BillTo }}{{ .Name }}{{ end }}`, "BillTo"},
		{"range condition", `{{ range .ChargeRows }}x{{ end }}`, "ChargeRows"},
		{"nested if inside range", `{{ range .ChargeRows }}{{ if .Amount }}x{{ end }}{{ end }}`, "ChargeRows.Amount"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, diags := engine.Parse(&ParseRequest{
				Field:        "bodyHtml",
				Channel:      ChannelPDF,
				Source:       tt.source,
				AllowedPaths: invoicePaths(),
				Strict:       true,
			})
			require.False(t, diags.HasErrors(), diags.Summary())
			assert.Contains(t, compiled.Paths, tt.want)
		})
	}
}

func TestNestedTemplatesAreRejected(t *testing.T) {
	engine := testEngine(t)

	tests := []struct {
		name   string
		source string
	}{
		{"define", `{{ define "row" }}<tr></tr>{{ end }}<table></table>`},
		{"block", `{{ block "row" . }}<tr></tr>{{ end }}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, diags := engine.Parse(&ParseRequest{
				Field:   "bodyHtml",
				Channel: ChannelPDF,
				Source:  tt.source,
			})
			require.True(t, diags.HasErrors())
			assert.True(t, hasCode(diags, CodeNestedTemplateForbidden))
		})
	}
}

func TestUnknownFunctionIsRejected(t *testing.T) {
	engine := testEngine(t)

	// These are the names an author would reach for to escape auto-escaping.
	for _, name := range ForbiddenFunctionNames() {
		if name == "template" {
			// {{template}} is an action, not a call; it has its own diagnostic.
			continue
		}
		t.Run(name, func(t *testing.T) {
			_, diags := engine.Parse(&ParseRequest{
				Field:        "bodyHtml",
				Channel:      ChannelPDF,
				Source:       `<p>{{ ` + name + ` .Memo }}</p>`,
				AllowedPaths: invoicePaths(),
			})
			require.True(t, diags.HasErrors())
			assert.True(t, hasCode(diags, CodeUnknownFunction), "codes: %v", codes(diags))
		})
	}
}

// TestBuiltinEscapersAreRejected is the case a FuncMap cannot cover. Go installs
// html, js, urlquery, and printf as builtins, so they parse regardless of what we
// define. None of them can produce unescaped output, but they let an author choose
// an escaping context, and that decision belongs to contextual auto-escaping.
func TestBuiltinEscapersAreRejected(t *testing.T) {
	engine := testEngine(t)

	for _, name := range []string{"html", "js", "urlquery", "printf"} {
		t.Run(name, func(t *testing.T) {
			source := `<p>{{ html .Memo }}</p>`
			if name == "printf" {
				source = `<p>{{ printf "%s" .Memo }}</p>`
			} else {
				source = `<p>{{ ` + name + ` .Memo }}</p>`
			}

			_, diags := engine.Parse(&ParseRequest{
				Field:        "bodyHtml",
				Channel:      ChannelPDF,
				Source:       source,
				AllowedPaths: invoicePaths(),
			})
			require.True(t, diags.HasErrors(),
				"%s is a builtin and must still be refused", name)
			assert.True(t, hasCode(diags, CodeUnknownFunction), "codes: %v", codes(diags))
		})
	}
}

func TestPermittedBuiltinsStillWork(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:   "bodyHtml",
		Channel: ChannelPDF,
		Source: `{{ if and (gt (len .ChargeRows) 0) (ne .InvoiceNumber "") }}` +
			`<p>{{ index .ChargeRows 0 }}</p>{{ end }}`,
		AllowedPaths: invoicePaths(),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	_, err := engine.Render(t.Context(), compiled, newInvoiceSample())
	require.NoError(t, err)
}

func TestTemplateSizeCap(t *testing.T) {
	engine, err := New(Config{MaxSourceBytes: 64})
	require.NoError(t, err)

	_, diags := engine.Parse(&ParseRequest{
		Field:   "bodyHtml",
		Channel: ChannelPDF,
		Source:  strings.Repeat("<p>x</p>", 100),
	})
	require.True(t, diags.HasErrors())
	assert.True(t, hasCode(diags, CodeTemplateTooLarge))
}

// TestVerifyRejectsUnsafeInterpolationContext covers the case where the escaper
// cannot decide how to escape, and would otherwise emit ZgotmplZ into a customer
// document.
func TestVerifyRejectsUnsafeInterpolationContext(t *testing.T) {
	engine := testEngine(t)

	// A free-text field used as a CSS value: html/template cannot escape it, so
	// it substitutes a placeholder rather than letting the declaration be closed
	// early. Real memos contain punctuation, so this breaks on real data.
	source := `<style>body{color:{{ .Memo }}}</style>`
	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       source,
		AllowedPaths: invoicePaths(),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	sample := newInvoiceSample()
	sample.Memo = "red}body{background:url(http://evil.example/x)"

	verifyDiags := engine.Verify(t.Context(), VerifyRequest{
		Field:    "bodyHtml",
		Channel:  ChannelPDF,
		Compiled: compiled,
		Sample:   sample,
	})
	require.True(t, verifyDiags.HasErrors())
	assert.True(t, hasCode(verifyDiags, CodeUnsafeInterpolationContext))
}

func TestVerifyAcceptsASoundTemplate(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<h1>{{ .InvoiceNumber }}</h1><p>{{ money .CurrencyCode .Total }}</p>`,
		AllowedPaths: invoicePaths(),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	verifyDiags := engine.Verify(t.Context(), VerifyRequest{
		Field:    "bodyHtml",
		Channel:  ChannelPDF,
		Compiled: compiled,
		Sample:   newInvoiceSample(),
	})
	assert.False(t, verifyDiags.HasErrors(), verifyDiags.Summary())
}

func TestVerifyRejectsATemplateThatFailsOnSampleData(t *testing.T) {
	engine := testEngine(t)

	// Calling a method that does not exist parses but cannot execute.
	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<p>{{ .Total.NoSuchMethod }}</p>`,
		AllowedPaths: append(invoicePaths(), "Total.NoSuchMethod"),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	verifyDiags := engine.Verify(t.Context(), VerifyRequest{
		Field:    "bodyHtml",
		Channel:  ChannelPDF,
		Compiled: compiled,
		Sample:   newInvoiceSample(),
	})
	require.True(t, verifyDiags.HasErrors())
	assert.True(t, hasCode(verifyDiags, CodeSampleRenderFailed))
}

// TestDataFlowingThroughATemplateIsEscaped is the property that makes an
// organization-authored template safe even when the *data* is hostile: a customer
// name is attacker-influenced, and it must not become markup.
func TestDataFlowingThroughATemplateIsEscaped(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<div>{{ .BillTo.Name }}</div><p>{{ .Memo }}</p>`,
		AllowedPaths: invoicePaths(),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	sample := newInvoiceSample()
	sample.BillTo.Name = `<script>alert(1)</script>`
	sample.Memo = `" onmouseover="alert(1)`

	out, err := engine.Render(t.Context(), compiled, sample)
	require.NoError(t, err)

	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
	assert.NotContains(t, out, `onmouseover="alert(1)"`)
}

func TestNl2brEscapesBeforeAddingMarkup(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<div>{{ nl2br .Memo }}</div>`,
		AllowedPaths: invoicePaths(),
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	sample := newInvoiceSample()
	sample.Memo = "line one\n<script>alert(1)</script>"

	out, err := engine.Render(t.Context(), compiled, sample)
	require.NoError(t, err)

	assert.Contains(t, out, "line one<br>")
	assert.NotContains(t, out, "<script>")
	assert.Contains(t, out, "&lt;script&gt;")
}

func TestTextChannelHasNoNl2br(t *testing.T) {
	engine := testEngine(t)

	_, diags := engine.Parse(&ParseRequest{
		Field:        "bodyText",
		Channel:      ChannelEmailText,
		Source:       `{{ nl2br .Memo }}`,
		AllowedPaths: invoicePaths(),
	})
	require.True(t, diags.HasErrors())
	assert.True(t, hasCode(diags, CodeUnknownFunction))
}

func TestTextChannelSkipsHTMLSanitizing(t *testing.T) {
	engine := testEngine(t)

	// A plain-text body that mentions a tag is text, not markup.
	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyText",
		Channel:      ChannelEmailText,
		Source:       "Invoice {{ .InvoiceNumber }}\nDo not paste <script> anywhere.",
		AllowedPaths: invoicePaths(),
		Strict:       true,
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	out, err := engine.Render(t.Context(), compiled, newInvoiceSample())
	require.NoError(t, err)
	assert.Contains(t, out, "<script>", "text output must not be HTML-escaped")
}

func TestOutputSizeCapAborts(t *testing.T) {
	engine, err := New(Config{MaxHTMLBytes: 2048})
	require.NoError(t, err)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `{{ range .Rows }}<p>{{ . }}</p>{{ end }}`,
		AllowedPaths: []string{"Rows"},
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	rows := make([]string, 100000)
	for i := range rows {
		rows[i] = "a fairly long row of text to burn through the budget"
	}

	_, err = engine.Render(t.Context(), compiled, map[string]any{"Rows": rows})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOutputTooLarge)
}

// TestRenderHonoursCancellation confirms a long render stops when the caller
// gives up, and that nothing is left running behind it.
func TestRenderHonoursCancellation(t *testing.T) {
	defer goleak.VerifyNone(t)

	engine, err := New(Config{MaxHTMLBytes: 1 << 30})
	require.NoError(t, err)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `{{ range .Rows }}<p>{{ . }}</p>{{ end }}`,
		AllowedPaths: []string{"Rows"},
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	rows := make([]string, 500000)
	for i := range rows {
		rows[i] = "row"
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	_, err = engine.Render(ctx, compiled, map[string]any{"Rows": rows})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestMissingKeyIsAnErrorNotABlank(t *testing.T) {
	engine := testEngine(t)

	compiled, diags := engine.Parse(&ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<p>{{ .Absent }}</p>`,
		AllowedPaths: []string{"Absent"},
	})
	require.False(t, diags.HasErrors(), diags.Summary())

	// A map context with the key absent must fail loudly rather than render a
	// blank where a number belongs.
	_, err := engine.Render(t.Context(), compiled, map[string]any{})
	require.Error(t, err)
}

func TestCacheReturnsTheSameCompilation(t *testing.T) {
	engine := testEngine(t)

	req := &ParseRequest{
		Field:        "bodyHtml",
		Channel:      ChannelPDF,
		Source:       `<p>{{ .InvoiceNumber }}</p>`,
		AllowedPaths: invoicePaths(),
		CacheKey:     "version-abc:PDF",
	}

	first, diags := engine.Parse(req)
	require.False(t, diags.HasErrors())
	second, _ := engine.Parse(req)
	assert.Same(t, first, second, "a published version is immutable and should compile once")

	engine.Invalidate("version-abc:PDF")
	third, _ := engine.Parse(req)
	assert.NotSame(t, first, third, "invalidation must force a recompile")
}

func TestContentHashIsStableAndDistinct(t *testing.T) {
	a := ContentHash("<p>a</p>")
	b := ContentHash("<p>a</p>")
	c := ContentHash("<p>b</p>")

	assert.Equal(t, a, b)
	assert.NotEqual(t, a, c)
	assert.Len(t, a, 64)
}

func TestMaxOutputBytesByChannel(t *testing.T) {
	engine, err := New(Config{MaxHTMLBytes: 1000, MaxTextBytes: 100})
	require.NoError(t, err)

	assert.Equal(t, int64(1000), engine.MaxOutputBytes(ChannelPDF))
	assert.Equal(t, int64(1000), engine.MaxOutputBytes(ChannelEmailHTML))
	assert.Equal(t, int64(100), engine.MaxOutputBytes(ChannelEmailText))
	assert.Equal(t, int64(100), engine.MaxOutputBytes(ChannelSubject))
	assert.Equal(t, int64(100), engine.MaxOutputBytes(ChannelNotificationTitle))
	assert.Equal(t, int64(100), engine.MaxOutputBytes(ChannelNotificationBody))
}
