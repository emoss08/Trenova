package modeladaptertest

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// hostRouter sends requests for the named hosts to a test server instead, so
// a token endpoint the code always calls by its real address can be faked.
type HostRouter struct {
	Routes map[string]*url.URL
}

func (h HostRouter) RoundTrip(req *http.Request) (*http.Response, error) {
	if target, ok := h.Routes[req.URL.Host]; ok {
		routed := req.Clone(req.Context())
		routed.URL.Scheme = target.Scheme
		routed.URL.Host = target.Host
		routed.Host = target.Host
		return http.DefaultTransport.RoundTrip(routed)
	}

	return http.DefaultTransport.RoundTrip(req)
}

type TokenServer struct {
	Server   *httptest.Server
	Requests atomic.Int32
	Mu       sync.Mutex
	Forms    []url.Values
	Paths    []string
}

func (ts *TokenServer) URL(t *testing.T) *url.URL {
	t.Helper()

	parsed, err := url.Parse(ts.Server.URL)
	require.NoError(t, err)

	return parsed
}
