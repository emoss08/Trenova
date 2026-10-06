package routelint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	apiDir      = ".."
	handlersDir = "../handlers"
)

func TestProtectedRoutesAreDiscovered(t *testing.T) {
	t.Parallel()

	routes, err := ProtectedRoutes(apiDir, handlersDir)
	require.NoError(t, err)
	require.NotEmpty(t, routes)

	byPackage := make(map[string]int)
	for _, route := range routes {
		require.NotEmpty(t, route.Method)
		require.True(
			t,
			strings.HasPrefix(route.Path, APIV1BasePath),
			"route %s must live under %s", route.Key(), APIV1BasePath,
		)
		byPackage[route.Package]++
	}

	for _, pkg := range []string{"workerhandler", "shipmenthandler", "iamhandler"} {
		require.NotZerof(t, byPackage[pkg], "expected routes discovered for %s", pkg)
	}
}

func TestJoinPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		base     string
		relative string
		want     string
	}{
		{base: "/api/v1", relative: "/workers", want: "/api/v1/workers"},
		{base: "/api/v1", relative: "/workers/", want: "/api/v1/workers/"},
		{base: "/api/v1/workers", relative: "/", want: "/api/v1/workers/"},
		{
			base:     "/api/v1/portal/",
			relative: "loads/:id/documents/",
			want:     "/api/v1/portal/loads/:id/documents/",
		},
		{base: "/api/v1/workers", relative: "", want: "/api/v1/workers"},
		{base: "/api/v1", relative: "/workers/:workerID/", want: "/api/v1/workers/:workerID/"},
	}

	for _, tt := range tests {
		t.Run(tt.base+"|"+tt.relative, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, joinPaths(tt.base, tt.relative))
		})
	}
}

func writeLintFixture(t *testing.T, routerBody string, handlerBody string) (string, string) {
	t.Helper()

	root := t.TempDir()
	apiPath := filepath.Join(root, "api")
	handlerPath := filepath.Join(root, "api", "handlers", "demohandler")
	require.NoError(t, os.MkdirAll(handlerPath, 0o755))

	router := `package api

import "github.com/emoss08/trenova/internal/api/handlers/demohandler"

type Router struct {
	demoHandler *demohandler.Handler
}

func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	protected := r.protectedGroup(rg)
` + routerBody + `
}
`
	require.NoError(t, os.WriteFile(filepath.Join(apiPath, routerFileName), []byte(router), 0o600))

	handler := "package demohandler\n\n" + handlerBody
	require.NoError(
		t,
		os.WriteFile(filepath.Join(handlerPath, "handler.go"), []byte(handler), 0o600),
	)

	return apiPath, filepath.Join(root, "api", "handlers")
}

func TestProtectedRoutesRejectsUnknownDelegation(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.demoHandler.RegisterRoutes(protected)\n",
		`func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/demo")
	registerSomethingElse(api, h)
	api.GET("/", h.list)
}
`)

	_, err := ProtectedRoutes(apiPath, handlersPath)
	require.ErrorContains(t, err, "cannot follow")
	require.ErrorContains(t, err, "registerSomethingElse")
}

func TestProtectedRoutesRejectsMissingRegistrar(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.demoHandler.RegisterRoutes(protected)\n",
		`func (h *Handler) RegisterRoutes(rg gin.IRouter) {
	rg.GET("/demo/", h.list)
}
`)

	_, err := ProtectedRoutes(apiPath, handlersPath)
	require.ErrorContains(t, err, "declares no")
	require.ErrorContains(t, err, "RegisterRoutes")
}

func TestProtectedRoutesRejectsUnresolvableHandlerField(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.mysteryHandler.RegisterRoutes(protected)\n",
		`func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/demo/", h.list)
}
`)

	_, err := ProtectedRoutes(apiPath, handlersPath)
	require.ErrorContains(t, err, "cannot resolve to a handler package")
	require.ErrorContains(t, err, "mysteryHandler")
}

func TestProtectedRoutesSkipEditionRegistrations(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.demoHandler.RegisterRoutes(protected)\n\tr.editionRoutes.RegisterProtected(protected)\n",
		`func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/demo/", h.list)
}
`)

	routes, err := ProtectedRoutes(apiPath, handlersPath)
	require.NoError(t, err)
	require.Len(t, routes, 1)
	require.Equal(t, "GET /api/v1/demo/", routes[0].Key())
}

func TestProtectedRoutesFollowsDelegationToNestedGroups(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.demoHandler.RegisterRoutes(protected)\n",
		`func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	api := rg.Group("/demo")
	h.registerNested(api.Group("/nested"))
	api.GET("/", h.list)
}

func (h *Handler) registerNested(nested *gin.RouterGroup) {
	nested.POST("/deep/", h.create)
}
`)

	routes, err := ProtectedRoutes(apiPath, handlersPath)
	require.NoError(t, err)

	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, route.Key())
	}
	require.ElementsMatch(
		t,
		[]string{"GET /api/v1/demo/", "POST /api/v1/demo/nested/deep/"},
		keys,
	)
}

func TestProtectedRoutesFollowsChainedGroupRegistrations(t *testing.T) {
	t.Parallel()

	apiPath, handlersPath := writeLintFixture(t,
		"\tr.demoHandler.RegisterRoutes(protected)\n",
		`func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Group("/demo").GET("/", h.list)
	rg.Group("/demo").Group("/nested").POST("/deep/", h.create)
}
`)

	routes, err := ProtectedRoutes(apiPath, handlersPath)
	require.NoError(t, err)

	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, route.Key())
	}
	require.ElementsMatch(
		t,
		[]string{"GET /api/v1/demo/", "POST /api/v1/demo/nested/deep/"},
		keys,
		"a route registered directly on a chained group must still be discovered, "+
			"otherwise it ships unclassified and fails open",
	)
}
