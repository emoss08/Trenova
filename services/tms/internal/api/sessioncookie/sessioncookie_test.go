package sessioncookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetAndClear(t *testing.T) {
	t.Parallel()

	cfg := &config.SessionConfig{
		Name:     "trenova_session",
		Path:     "/",
		Secure:   true,
		HTTPOnly: true,
		SameSite: "lax",
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	Set(c, cfg, "token-value", time.Now().Add(time.Hour).Unix())

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "trenova_session", cookies[0].Name)
	assert.Equal(t, "token-value", cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	assert.InDelta(t, 3_600, cookies[0].MaxAge, 5)

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	Clear(c, cfg)
	cookies = recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Empty(t, cookies[0].Value)
	assert.Negative(t, cookies[0].MaxAge)
}
