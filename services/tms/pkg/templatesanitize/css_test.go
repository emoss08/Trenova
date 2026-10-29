package templatesanitize

import (
	"testing"

	"github.com/emoss08/trenova/pkg/templatediag"
	"github.com/emoss08/trenova/pkg/templatediagtest"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeCSSRejectsStylesheetCorpus(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"style end tag":             `p { color: red } </style><script>alert(1)</script>`,
		"uppercase style end":       `p {} </STYLE >`,
		"style end with space":      `p {} </ style>`,
		"import":                    `@import "https://evil.example/x.css";`,
		"escaped import":            `@\69mport "https://evil.example/x.css";`,
		"commented import":          `@im/**/port "x.css";`,
		"remote url":                `body { background: url(https://evil.example/x.png) }`,
		"quoted remote url":         `body { background: url("//evil.example/x.png") }`,
		"javascript url":            `a { background: url('javascript:alert(1)') }`,
		"escaped url scheme":        `a { background: url(h\74tps://evil.example/) }`,
		"expression":                `p { width: expression(alert(1)) }`,
		"commented expression":      `p { width: expr/**/ession(alert(1)) }`,
		"escaped expression":        `p { width: \65xpression(alert(1)) }`,
		"behavior":                  `p { behavior: url(x.htc) }`,
		"escaped behavior":          `p { beh\61vior: url(x.htc) }`,
		"moz binding":               `p { -moz-binding: url(x.xml#x) }`,
		"image-set remote":          `p { background-image: image-set("https://evil.example/a.png" 1x) }`,
		"webkit image-set remote":   `p { background-image: -webkit-image-set('//evil.example/a.png' 1x) }`,
		"escape across a backslash": `@\000069mport "x.css";`,
	}

	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diags := CSS("cssContent", source)
			assert.True(
				t,
				templatediagtest.HasCode(diags, templatediag.CodeForbiddenCSS),
				"%s: %v",
				name,
				diags,
			)
			assert.True(t, diags.HasErrors(), name)
		})
	}
}

func TestSanitizeCSSAcceptsAnOrdinaryStylesheet(t *testing.T) {
	t.Parallel()

	source := `
		/* invoice layout */
		@page { size: letter; margin: 0.5in }
		body { font-family: "Inter", sans-serif; color: #171717 }
		.logo { background: url("data:image/png;base64,iVBORw0KGgo=") no-repeat }
		.mark::before { content: "\2014"; }
		table td { padding: 4px 8px }
	`
	assert.Empty(t, CSS("cssContent", source))
}
