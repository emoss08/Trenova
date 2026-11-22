package emailpreview

import (
	"html"
	"regexp"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templateregistry"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	doctype       = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN"`
	band          = `bgcolor="#ffa31a"`
	button        = `bgcolor="#121210"`
	highlight     = `background-color:#ffa31a;color:#141413;padding:0 6px`
	footerCell    = `font-size:13px;line-height:1.6;color:#55544e"><p style="margin:0;padding:0">`
	preheaderOpen = `<div style="margin:0;padding:0;display:none;`
)

var (
	placeholder      = regexp.MustCompile(`\{\d+[,}]`)
	h1Pattern        = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	titlePattern     = regexp.MustCompile(`<title>(.*?)</title>`)
	preheaderPattern = regexp.MustCompile(
		`(?s)` + regexp.QuoteMeta(preheaderOpen) + `[^>]*>(.*?)</div>`,
	)
)

func TestEveryEmailKindHasAPreview(t *testing.T) {
	t.Parallel()

	cases, err := Catalog()
	require.NoError(t, err)

	covered := make(map[documenttemplate.Kind]bool, len(cases))
	for _, c := range cases {
		covered[c.Kind] = true
	}
	for _, def := range templateregistry.Default().All() {
		if !def.HasChannel(documenttemplate.ChannelEmailHTML) {
			continue
		}
		assert.True(t, covered[def.Kind], "%s sends email but has no preview", def.Kind)
	}
}

func TestEveryEmailFollowsTheStandard(t *testing.T) {
	t.Parallel()

	cases, err := Catalog()
	require.NoError(t, err)
	renderer := NewRenderer()

	for _, c := range cases {
		for _, locale := range i18n.Supported() {
			t.Run(c.Slug+"/"+string(locale), func(t *testing.T) {
				t.Parallel()

				out, renderErr := renderer.Render(t.Context(), c, locale)
				require.NoError(t, renderErr)
				assertDocument(t, out, locale)
				assertHeader(t, out, c)
				assertOneHeading(t, out)
				assertAtMostOneAction(t, out)
				assertFooter(t, out)
				assertText(t, out)
			})
		}
	}
}

func TestEveryEmailSurvivesHostileData(t *testing.T) {
	t.Parallel()

	cases, err := Catalog()
	require.NoError(t, err)
	renderer := NewRenderer()
	registry := templateregistry.Default()

	seen := make(map[documenttemplate.Kind]bool, len(cases))
	for _, c := range cases {
		if seen[c.Kind] {
			continue
		}
		seen[c.Kind] = true

		probe, probeErr := registry.HostileProbeContext(c.Kind)
		require.NoError(t, probeErr)
		hostile := *c
		hostile.Data = probe

		t.Run(string(c.Kind), func(t *testing.T) {
			t.Parallel()

			out, renderErr := renderer.Render(t.Context(), &hostile, i18n.Default)
			require.NoError(t, renderErr)
			assert.NotContains(t, out.HTML, "<script")
			assert.NotContains(t, out.HTML, "ZgotmplZ")
			assert.LessOrEqual(t, strings.Count(out.HTML, band), 1)
		})
	}
}

func assertDocument(t *testing.T, out *Rendered, locale i18n.Locale) {
	t.Helper()

	assert.True(t, strings.HasPrefix(out.HTML, doctype), "the email is a full XHTML document")
	assert.Contains(t, out.HTML, `<html dir="ltr" lang="`+string(locale)+`">`)
	for _, meta := range []string{
		`<meta content="width=device-width" name="viewport" />`,
		`<meta content="text/html; charset=UTF-8" http-equiv="Content-Type" />`,
		`<meta name="x-apple-disable-message-reformatting" />`,
		`<meta content="telephone=no,address=no,email=no,date=no,url=no" name="format-detection" />`,
	} {
		assert.Contains(t, out.HTML, meta)
	}

	title := titlePattern.FindStringSubmatch(out.HTML)
	require.Len(t, title, 2)
	assert.Equal(t, out.Subject, html.UnescapeString(title[1]), "the title is the subject")
	assert.NotEmpty(t, strings.TrimSpace(out.Subject))

	assert.Equal(
		t,
		1,
		strings.Count(out.HTML, "<style"),
		"one style block, for the reset and mobile rules",
	)
	assert.Contains(t, out.HTML, ":root { color-scheme: light only; }")
	assert.Contains(t, out.HTML, "@media (max-width: 620px)")
	assert.NotContains(t, out.HTML, "prefers-color-scheme", "the email is light only")

	preheaders := preheaderPattern.FindAllStringSubmatch(out.HTML, -1)
	require.Len(t, preheaders, 1, "exactly one preheader")
	text := strings.TrimSpace(html.UnescapeString(preheaders[0][1]))
	assert.NotEmpty(t, text)
	assert.NotEqual(
		t,
		strings.TrimSpace(out.Subject),
		text,
		"the preheader never repeats the subject",
	)
	assert.True(t, strings.Index(out.HTML, preheaderOpen) < strings.Index(out.HTML, "<table"),
		"the preheader is the first thing in the body")

	assert.NotContains(t, out.HTML, "<hl>")
	assert.NotContains(t, out.HTML, "ZgotmplZ")
	assert.False(t, placeholder.MatchString(out.HTML), "a placeholder was left unfilled")
	assert.NotContains(t, out.HTML, "Trenova Logistics")
}

func assertHeader(t *testing.T, out *Rendered, c *Case) {
	t.Helper()

	assert.Contains(t, out.HTML, html.EscapeString(c.Brand.Name), "the header names the company")
	if c.Brand.Mark == "" {
		assert.NotContains(t, out.HTML, "<img", "with no logo there is no image at all")
		return
	}
	assert.Equal(t, 1, strings.Count(out.HTML, "<img"), "the company mark is the only image")
	assert.Contains(t, out.HTML, `<img alt="" width="30" height="30"`)
	assert.Contains(t, out.HTML, "border-radius:7px")
}

func assertOneHeading(t *testing.T, out *Rendered) {
	t.Helper()

	headings := h1Pattern.FindAllStringSubmatch(out.HTML, -1)
	require.Len(t, headings, 1, "exactly one heading")
	assert.Equal(t, 1, strings.Count(headings[0][1], highlight),
		"exactly one highlighted phrase in the heading")
	assert.Equal(
		t,
		1,
		strings.Count(out.HTML, highlight),
		"the highlight is used only in the heading",
	)
}

func assertAtMostOneAction(t *testing.T, out *Rendered) {
	t.Helper()

	bands := strings.Count(out.HTML, band)
	buttons := strings.Count(out.HTML, button)
	assert.LessOrEqual(t, bands, 1, "an email has at most one action band")
	assert.Equal(t, bands, buttons, "the one button lives in the band, and only there")
	if bands == 0 {
		return
	}
	start := strings.Index(out.HTML, band)
	end := start + strings.Index(out.HTML[start:], "</td></tr>")
	inside := out.HTML[start:end]
	assert.Contains(t, inside, button, "the button is inside the band")
	assert.Contains(t, inside, `style="margin:0;padding:14px 28px;background-color:#121210"`,
		"the button pads its cell, never a bare link")
}

func assertFooter(t *testing.T, out *Rendered) {
	t.Helper()

	assert.Equal(t, 1, strings.Count(out.HTML, footerCell), "one footer, below the card")
	footerAt := strings.Index(out.HTML, footerCell)
	cardAt := strings.Index(out.HTML, "border:1px solid #121210")
	assert.Greater(t, footerAt, cardAt, "the footer sits under the card")
}

func assertText(t *testing.T, out *Rendered) {
	t.Helper()

	text := strings.TrimSpace(out.Text)
	require.NotEmpty(t, text, "every email keeps its plain-text part")
	assert.NotContains(t, text, "<hl>")
	assert.NotContains(t, text, "{{")
	assert.False(t, placeholder.MatchString(text), "a placeholder was left unfilled")

	idx := strings.LastIndex(text, "\n--\n")
	require.Positive(t, idx, "the text part ends with its footer line")
	footerLine := strings.TrimSpace(text[idx+4:])
	assert.NotEmpty(t, footerLine)

	footerAt := strings.Index(out.HTML, footerCell)
	require.Positive(t, footerAt)
	firstLine := strings.SplitN(footerLine, "\n", 2)[0]
	assert.Contains(t, html.UnescapeString(out.HTML[footerAt:]), firstLine,
		"the text part says the same footer line as the HTML")
}
