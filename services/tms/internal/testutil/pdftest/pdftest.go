package pdftest

import (
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/pdfreader"
	"github.com/stretchr/testify/require"
)

var shared = sync.OnceValue(func() *pdfreader.Reader {
	return pdfreader.NewReader(nil, nil)
})

func Reader() services.PDFReader {
	return shared()
}

func Open(t testing.TB, pdf []byte) services.PDFDocument {
	t.Helper()

	doc, err := shared().Open(t.Context(), pdf)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, doc.Close()) })

	return doc
}

func PageTexts(t testing.TB, pdf []byte) []string {
	t.Helper()

	doc := Open(t, pdf)
	texts := make([]string, 0, doc.PageCount())
	for page := range doc.PageCount() {
		text, err := doc.PageText(t.Context(), page)
		require.NoError(t, err)
		texts = append(texts, text)
	}

	return texts
}

func Text(t testing.TB, pdf []byte) string {
	t.Helper()
	return strings.Join(PageTexts(t, pdf), "\n")
}

func PageCount(t testing.TB, pdf []byte) int {
	t.Helper()
	return Open(t, pdf).PageCount()
}
