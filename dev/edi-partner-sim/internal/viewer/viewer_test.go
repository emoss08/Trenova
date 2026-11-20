package viewer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTheViewerFilesAreServedWithTheirTypes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(Handler())
	defer server.Close()
	for _, c := range []struct {
		path, contentType, contains string
	}{
		{"/", "text/html", `<script src="viewer.js" defer></script>`},
		{"/viewer.css", "text/css", "--trenova: #1d7a72"},
		{"/viewer.js", "text/javascript", "EventSource"},
		{"/ladder.js", "text/javascript", "Sim.drawLadder"},
		{"/inspector.js", "text/javascript", "Sim.renderInspector"},
	} {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
			server.URL+c.path, nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("GET %s: %v", c.path, err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: %d", c.path, response.StatusCode)
			continue
		}
		if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, c.contentType) {
			t.Errorf("GET %s: Content-Type %q, want %s", c.path, got, c.contentType)
		}
		if response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("GET %s: no nosniff header", c.path)
		}
		if !strings.Contains(string(body), c.contains) {
			t.Errorf("GET %s lacks %q", c.path, c.contains)
		}
	}
}
