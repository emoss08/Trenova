package authhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func decodeCSRF(t *testing.T, w *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

func TestCSRFTokenWithoutSessionAnswersAnEmptyToken(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	_, router := newTestHandler(svc)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/csrf", nil))

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeCSRF(t, w)
	assert.Empty(t, body["csrfToken"])
	assert.Equal(t, "X-CSRF-Token", body["headerName"])
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestCSRFTokenWithAStaleSessionAnswersAnEmptyToken(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	svc.EXPECT().AuthenticateSession(mock.Anything, "stale").Return(nil, errors.New("expired"))
	_, router := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/csrf", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "stale"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, decodeCSRF(t, w)["csrfToken"])
}

func TestCSRFTokenWithASessionIsBoundToIt(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")
	svc := mocks.NewMockAuthService(t)
	svc.EXPECT().AuthenticateSession(mock.Anything, "live").Return(&session.Session{ID: sessionID}, nil)
	_, router := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/csrf", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "live"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, csrf.Token(sessionID.String(), "test-session-secret"), decodeCSRF(t, w)["csrfToken"])
}
