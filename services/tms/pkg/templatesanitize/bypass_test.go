package templatesanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTML_RefusesWhatUsedToSlipPast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source string
	}{
		{name: "a self-closed style", source: `<style/>@import "http://x/a";`},
		{name: "a quote inside a url", source: `<p style="background:url('http://x/a"b')"></p>`},
		{name: "an escaped template marker", source: `<p style="background:url(http://x/\\7b\\7b)"></p>`},
		{name: "a bracket inside image-set", source: `<p style="background:image-set('http://x/a)b' 1x)"></p>`},
		{
			name:   "a remote second srcset candidate",
			source: `<img srcset="data:image/png;base64,AAAA 1x, http://x/a.png 2x">`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, HTML("body", tt.source, DocumentOptions()).HasErrors())
		})
	}
}

func TestCSS_RefusesWhatHidesBehindACommentMarkerInAString(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"an import": `a{content:"/*"}@import url(http://x/a);b{content:"*/"}`,
		"a fetch":   `a{content:'/*'}b{background:url(http://x/a)}c{content:'*/'}`,
		"no close":  `a{content:"/*"}@import url(http://x/a);`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, CSS("cssContent", source, DocumentOptions()).HasErrors())
		})
	}
}

func TestCSS_StillDropsCommentsAroundStrayQuotes(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]string{
		"a quote in an unquoted url":  `a{background:url(it's)}b{width:expr/**/ession(alert(1))}`,
		"a string ended by a newline": "a{content:'open\n}b{width:expr/**/ession(alert(1))}",
		"an escaped quote":            `a{content:'it\'s'}b{width:expr/**/ession(alert(1))}`,
		"an escaped url name":         `a{background:u\72l(it's)}b{width:expr/**/ession(alert(1))}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, CSS("cssContent", source, DocumentOptions()).HasErrors())
		})
	}
}

func TestCSS_ReadsACommentMarkerInsideAStringAsText(t *testing.T) {
	t.Parallel()

	assert.False(t, CSS("cssContent", `a{content:"/* note */"}`, DocumentOptions()).HasErrors())
	assert.False(t, CSS("cssContent", `/* "quoted" */a{color:red}`, DocumentOptions()).HasErrors())
}

func TestHTML_ChecksAStyleElementLeftOpenAtTheEndOfTheSource(t *testing.T) {
	t.Parallel()

	for _, opts := range []Options{DocumentOptions(), RenderedDocumentOptions()} {
		assert.True(t, HTML("body", `<style>@import "http://x/a";`, opts).HasErrors())
		assert.True(t, HTML("body", `<p>x</p><style>a{b:url(http://x/a)}`, opts).HasErrors())
		assert.False(t, HTML("body", `<style>a{color:red}`, opts).HasErrors())
	}
}

func TestRenderedOutput_RefusesATemplateMarkerInAStylesheetFetch(t *testing.T) {
	t.Parallel()

	source := `a{background:url(http://x/{{y)}`

	assert.False(t, CSS("cssContent", source, DocumentOptions()).HasErrors())
	assert.True(t, CSS("cssContent", source, RenderedDocumentOptions()).HasErrors())
	assert.True(
		t,
		HTML("body", `<p style="background:url(http://x/{{y)"></p>`, RenderedDocumentOptions()).
			HasErrors(),
	)
	assert.True(
		t,
		HTML("body", `<style>`+source+`</style>`, RenderedDocumentOptions()).HasErrors(),
	)
}

func TestCSS_LengthChangingCharactersDoNotSkewTheFetchCheck(t *testing.T) {
	t.Parallel()

	kelvin := strings.Repeat("K", 11)
	source := `a{--x:` + kelvin + `data:,;background:url(http://x/a)}`

	assert.True(t, CSS("cssContent", source, DocumentOptions()).HasErrors())
	assert.True(t, HTML("body", `<style>`+source+`</style>`, DocumentOptions()).HasErrors())
}

func TestCSS_MatchesFetchFunctionsInAnyASCIICase(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		`a{background:URL(http://x/a)}`,
		`a{background:Url(http://x/a)}`,
		`@font-face{src:SRC("http://x/a")}`,
		`a{background:IMAGE-SET("http://x/a" 1x)}`,
	} {
		assert.True(t, CSS("cssContent", source, DocumentOptions()).HasErrors(), source)
	}
}

func TestHTML_StillAcceptsInlineImagesAndTemplateURLs(t *testing.T) {
	t.Parallel()

	for _, source := range []string{
		`<img srcset="data:image/png;base64,AAAA 1x, data:image/png;base64,BBBB 2x">`,
		`<img src="{{ .Logo }}">`,
		`<p style="background:url(data:image/png;base64,AAAA)"></p>`,
		`<p style="background:url({{ .Background }})"></p>`,
	} {
		assert.False(t, HTML("body", source, DocumentOptions()).HasErrors(), source)
	}
}

func TestRenderedOutput_HoldsFetchesAndLeavesLinksAlone(t *testing.T) {
	t.Parallel()

	options := RenderedDocumentOptions()

	assert.True(t, HTML("body", `<img src="https://tracker.example/a.png">`, options).HasErrors())
	assert.True(t, HTML("body", `<img src="{{ left over }}">`, options).HasErrors())
	assert.True(t, HTML("body", `<a href="javascript:alert(1)">x</a>`, options).HasErrors())
	assert.False(t, HTML("body", `<a href="/shipments/1">open</a>`, options).HasErrors())
}

func TestRenderedOutput_AcceptsTheDocumentSkeletonsStylesheet(t *testing.T) {
	t.Parallel()

	source := `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Invoice</title>` +
		`<style>@font-face{src:url(data:font/woff2;base64,AAAA)}</style></head><body>x</body></html>`

	assert.False(t, HTML("body", source, RenderedDocumentOptions()).HasErrors())
}
