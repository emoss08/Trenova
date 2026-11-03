package templateengine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapDocumentLeavesOutAStylesheetWithFindings(t *testing.T) {
	t.Parallel()

	document := WrapDocument(WrapRequest{
		Title:    "Invoice",
		BodyHTML: "<p>Total</p>",
		CSS:      `p { color: red } </style><script>alert(1)</script>`,
	})
	assert.NotContains(t, document, "<script>")
	assert.NotContains(t, document, "color: red")
	assert.Contains(t, document, "<p>Total</p>")

	clean := WrapDocument(WrapRequest{BodyHTML: "<p>Total</p>", CSS: `p { color: red }`})
	assert.Equal(t, 1, strings.Count(clean, "p { color: red }"))
}

func TestWrapDocumentDeclaresTheAppTypefacesWithoutApplyingThem(t *testing.T) {
	t.Parallel()

	document := WrapDocument(WrapRequest{BodyHTML: "<p>Total</p>"})

	assert.Equal(t, 2, strings.Count(document, `font-family:"Geist";`))
	assert.Equal(t, 2, strings.Count(document, `font-family:"Geist Mono";`))
	assert.Contains(t, document, "src:url(data:font/woff2;base64,")
	assert.Contains(t, document, `font-family: -apple-system, "Segoe UI"`)
}
