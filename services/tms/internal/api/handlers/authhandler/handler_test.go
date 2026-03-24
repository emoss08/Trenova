package authhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name:     "session_id",
				Path:     "/",
				Domain:   "localhost",
				Secure:   false,
				HTTPOnly: true,
			},
		},
	}
}

func newTestHandler(svc *mocks.MockAuthService) (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	cfg := newTestConfig()
	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})
	h := New(Params{
		Service:      svc,
		Logger:       zap.NewNop(),
		Config:       cfg,
		ErrorHandler: eh,
	})
	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)
	return h, r
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")
	sessionID := pulid.MustNew("ses_")

	svc := mocks.NewMockAuthService(t)
	svc.On("Login", mock.Anything, mock.Anything).Return(&services.LoginResponse{
		User: &tenant.User{
			ID:                    userID,
			BusinessUnitID:        buID,
			CurrentOrganizationID: orgID,
			EmailAddress:          "test@example.com",
		},
		ExpiresAt: 9999999999,
		SessionID: sessionID.String(),
	}, nil)

	_, r := newTestHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"emailAddress": "test@example.com",
		"password":     "password123",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp services.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, sessionID.String(), resp.SessionID)
	assert.Equal(t, int64(9999999999), resp.ExpiresAt)
	assert.Equal(t, "test@example.com", resp.User.EmailAddress)

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			found = true
			assert.Equal(t, sessionID.String(), c.Value)
			assert.True(t, c.HttpOnly)
			break
		}
	}
	assert.True(t, found, "session cookie should be set")
}

func TestLogin_InvalidJSON(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewReader([]byte("{invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.GreaterOrEqual(t, w.Code, 400)
}

func TestLogin_ServiceError(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("invalid credentials"))

	_, r := newTestHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"emailAddress": "test@example.com",
		"password":     "wrongpass",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.GreaterOrEqual(t, w.Code, 400)
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := mocks.NewMockAuthService(t)
	svc.On("Logout", mock.Anything, mock.Anything).Return(nil)

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertCalled(t, "Logout", mock.Anything, mock.Anything)

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, "", c.Value)
			assert.Less(t, c.MaxAge, 0)
			break
		}
	}
}

func TestLogout_NoCookie(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestValidateSession_Valid(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := mocks.NewMockAuthService(t)
	svc.On("ValidateSession", mock.Anything, mock.Anything).
		Return(&session.Session{ID: sessionID}, nil)

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["valid"])
}

func TestValidateSession_NoCookie(t *testing.T) {
	t.Parallel()

	svc := mocks.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["valid"])
}

func TestValidateSession_InvalidSession(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := mocks.NewMockAuthService(t)
	svc.On("ValidateSession", mock.Anything, mock.Anything).
		Return(nil, errors.New("session expired"))

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["valid"])
}
