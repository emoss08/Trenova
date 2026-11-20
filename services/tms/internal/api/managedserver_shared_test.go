//revive:disable-next-line:var-naming
package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/managedroutes"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type tenantRoutes struct {
	served int
}

func (c *tenantRoutes) RouteManaged(mux *http.ServeMux) {
	mux.HandleFunc("GET /managed/v1/tenants", func(w http.ResponseWriter, _ *http.Request) {
		c.served++
		w.WriteHeader(http.StatusOK)
	})
}

func sharedHandler(shared bool, routes *tenantRoutes) http.Handler {
	return api.NewManagedHandler(&api.ManagedHandlerParams{
		Usage:        &fakeManagedUsage{},
		Features:     newFeatures(&fakeFeatureRepo{}),
		Invites:      &fakeInvites{},
		Token:        managedToken,
		Logger:       zap.NewNop(),
		Shared:       shared,
		Contributors: []managedroutes.Contributor{routes},
	})
}

func call(t *testing.T, h http.Handler, method, target, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader("{}"))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestASharedServerRefusesTheDedicatedRoutes(t *testing.T) {
	t.Parallel()

	h := sharedHandler(true, &tenantRoutes{})
	for _, route := range []struct{ method, target string }{
		{http.MethodGet, "/managed/v1/readings?since=2026-11-20T00:00:00Z"},
		{http.MethodPut, "/managed/v1/state"},
		{http.MethodGet, "/managed/v1/features"},
		{http.MethodPut, "/managed/v1/features"},
		{http.MethodPut, "/managed/v1/first-administrator/invite"},
		{http.MethodGet, "/managed/v1/log-requests"},
		{http.MethodPut, "/managed/v1/log-requests/lrq_1"},
		{http.MethodPost, "/managed/v1/support/log-pulls"},
	} {
		rec := call(t, h, route.method, route.target, managedToken)
		assert.Equal(t, http.StatusConflict, rec.Code, route.method+" "+route.target)
		assert.Contains(t, rec.Body.String(), "/managed/v1/tenants")
	}
}

func TestContributedRoutesShareTheListenersAuthentication(t *testing.T) {
	t.Parallel()

	routes := &tenantRoutes{}
	h := sharedHandler(true, routes)

	rec := call(t, h, http.MethodGet, "/managed/v1/tenants", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	rec = call(t, h, http.MethodGet, "/managed/v1/tenants", strings.Repeat("x", len(managedToken)))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, 0, routes.served)

	rec = call(t, h, http.MethodGet, "/managed/v1/tenants", managedToken)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, routes.served)
}

func TestADedicatedInstanceKeepsItsRoutes(t *testing.T) {
	t.Parallel()

	h := sharedHandler(false, &tenantRoutes{})
	rec := call(t, h, http.MethodGet, "/managed/v1/features", managedToken)
	assert.Equal(t, http.StatusOK, rec.Code)
}
