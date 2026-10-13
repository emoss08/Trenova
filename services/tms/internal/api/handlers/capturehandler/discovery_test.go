package capturehandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/kernel/app/editioninfo"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discoveryHandler(t *testing.T, h *Handler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h.RegisterPublicRoutes(router.Group("/api/v1"))

	return router
}

func TestDiscoveryDescribesASelfHostedServer(t *testing.T) {
	t.Parallel()

	h := &Handler{
		edition: editioninfo.SelfHosted(),
		webBase: "https://tms.acme.test",
		version: "0.10.5",
	}
	recorder := httptest.NewRecorder()
	discoveryHandler(t, h).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/capture/discovery/", http.NoBody))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "public, max-age=300", recorder.Header().Get("Cache-Control"))

	var body Discovery
	require.NoError(t, sonic.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, Discovery{
		Product:       "trenova",
		Edition:       "community",
		SharedTenancy: false,
		Name:          "Trenova",
		Version:       "0.10.5",
		WebBaseURL:    "https://tms.acme.test",
		Pairing:       true,
	}, body)
}

func TestDiscoveryNamesTheCloudAndSaysWhenPairingIsOff(t *testing.T) {
	t.Parallel()

	h := &Handler{edition: editioninfo.NewStatic("cloud", true)}
	recorder := httptest.NewRecorder()
	discoveryHandler(t, h).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/capture/discovery/", http.NoBody))

	require.Equal(t, http.StatusOK, recorder.Code)

	var body Discovery
	require.NoError(t, sonic.Unmarshal(recorder.Body.Bytes(), &body))
	assert.Equal(t, "Trenova Cloud", body.Name)
	assert.Equal(t, "cloud", body.Edition)
	assert.True(t, body.SharedTenancy)
	assert.False(t, body.Pairing, "no web address means nothing to approve a computer at")
	assert.Empty(t, body.WebBaseURL)
}
