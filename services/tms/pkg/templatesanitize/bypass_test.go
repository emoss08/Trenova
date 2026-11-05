package templatesanitize

import (
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
