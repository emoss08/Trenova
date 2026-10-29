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
