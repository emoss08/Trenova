package capturehandler

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/services/capturereleaseservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func releaseHandler(t *testing.T, source string) *gin.Engine {
	t.Helper()
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	public, _ := key.Public().(ed25519.PublicKey)
	releases, err := capturereleaseservice.New(capturereleaseservice.Params{
		Logger: zap.NewNop(),
		Config: &config.Config{Update: config.UpdateConfig{
			CaptureManifestURL: source,
			CapturePublicKey:   base64.StdEncoding.EncodeToString(public),
		}},
	})
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	h := &Handler{
		releases: releases,
		l:        zap.NewNop(),
		eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: &config.Config{},
		}),
	}
	router := gin.New()
	h.RegisterPublicRoutes(router.Group("/api/v1"))

	return router
}

func TestLatestReleaseServesTheSignedManifestUnchanged(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("../../../core/services/capturereleaseservice/testdata/signed.json")
	require.NoError(t, err)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(source.Close)

	recorder := httptest.NewRecorder()
	releaseHandler(t, source.URL).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/capture/releases/latest/", http.NoBody))

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "public, max-age=300", recorder.Header().Get("Cache-Control"))
	assert.JSONEq(t, string(fixture), recorder.Body.String())
}

func TestLatestReleaseIsNotFoundWhenNothingIsPublished(t *testing.T) {
	t.Parallel()

	source := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(source.Close)

	recorder := httptest.NewRecorder()
	releaseHandler(t, source.URL).ServeHTTP(recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/capture/releases/latest/", http.NoBody))

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
