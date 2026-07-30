package documenttemplatehandler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// A template is markup an organization authored. The whole system rests on that
// markup never being executed with this application's origin: if it were, a
// template could read app-origin cookies and localStorage and the editor would
// become a privilege-escalation path for anyone who can edit a template.
//
// The rule is therefore that no endpoint serves template output as a document.
// GraphQL hands HTML back as a string for the client to put in a sandboxed
// iframe, and the one REST route serves a PDF. These tests hold that line.

func TestPreviewResponseIsAPDFAndCannotBeSniffedAsHTML(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/preview.pdf", nil)

	writePreviewPDF(c, "invoice.pdf", 3, []byte("%PDF-1.7\n"))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	// Without nosniff a browser may decide the body is HTML and render it as a
	// document on this origin, which is exactly what must not happen.
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
	require.NotContains(t, recorder.Header().Get("Content-Type"), "text/html")
}

func TestPreviewFilenameCannotBreakOutOfTheHeader(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/preview.pdf", nil)

	// Kinds are registry constants today. This asserts the filter that keeps
	// that from mattering if one ever becomes data.
	writePreviewPDF(c, "invoice\"\r\nSet-Cookie: a=b\r\n.pdf", 1, []byte("%PDF-1.7\n"))

	disposition := recorder.Header().Get("Content-Disposition")
	// What matters is that nothing can terminate the header or the quoted
	// string. The injected text surviving as inert characters inside the
	// filename is harmless — it never becomes a second header.
	require.NotContains(t, disposition, "\r")
	require.NotContains(t, disposition, "\n")
	require.NotContains(t, disposition, `"invoice"`)
	require.Equal(t, `inline; filename="invoice---Set-Cookie--a-b--.pdf-v1.pdf"`, disposition)
	require.Len(t, recorder.Header().Values("Set-Cookie"), 0)
}

func TestSanitizeFilenamePartKeepsOnlySafeCharacters(t *testing.T) {
	t.Parallel()

	require.Equal(t, "invoice.pdf", sanitizeFilenamePart("invoice.pdf"))
	require.Equal(t, "agent.request_missing_docs.email",
		sanitizeFilenamePart("agent.request_missing_docs.email"))
	require.Equal(t, "a-b-c", sanitizeFilenamePart("a/b\\c"))
}

// TestNoAPIEndpointServesTemplateOutputAsHTML scans the API surface rather than
// one handler, because the contract is about every endpoint: a future route that
// returns a rendered template as text/html would reintroduce the whole problem,
// and it would not be this package's test that caught it.
func TestNoAPIEndpointServesTemplateOutputAsHTML(t *testing.T) {
	t.Parallel()

	root := repoAPIRoot(t)

	// Two endpoints legitimately serve first-party HTML that no organization can
	// author: the API docs viewer and the GraphQL playground. Both ship markup
	// compiled into the binary. Anything else appearing here is a new endpoint
	// that must justify itself.
	allowed := map[string]string{
		filepath.Join(root, "handlers", "docshandler", "handler.go"): "Swagger UI shell",
		filepath.Join(root, "graphql", "handler.go"):                 "GraphQL playground shell",
	}

	var offenders []string
	for _, dir := range []string{"handlers", "graphql"} {
		err := filepath.WalkDir(
			filepath.Join(root, dir),
			func(path string, entry os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
					strings.HasSuffix(path, "_test.go") {
					return nil
				}

				source, readErr := os.ReadFile(path)
				if readErr != nil {
					return readErr
				}
				if _, ok := allowed[path]; ok {
					return nil
				}
				if strings.Contains(string(source), `"text/html`) {
					offenders = append(offenders, path)
				}

				return nil
			},
		)
		require.NoError(t, err)
	}

	require.Empty(t, offenders,
		"an API endpoint declares a text/html content type; organization-authored "+
			"template markup must never be served as a document on the app origin")
}

// TestPreviewHTMLIsDeclaredAsAStringInTheSchema pins the GraphQL side of the
// same contract: the field is a String the client must place in a sandboxed
// iframe, not a scalar that invites rendering.
func TestPreviewHTMLIsDeclaredAsAStringInTheSchema(t *testing.T) {
	t.Parallel()

	schemaPath := filepath.Join(
		repoAPIRoot(t), "graphql", "schema", "document_template.graphqls",
	)
	source, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	require.Contains(t, string(source), "type DocumentTemplatePreview {")
	require.Contains(t, string(source), "html: String!")
	require.NotContains(t, string(source), "html: HTML")
}

func repoAPIRoot(t *testing.T) string {
	t.Helper()

	// The test binary runs in its own package directory; the API root is two
	// levels up from internal/api/handlers/<pkg>.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(root, "handlers"))
	require.NoError(t, err)
	require.True(t, info.IsDir(), "expected to locate internal/api, got %s", root)

	return root
}
