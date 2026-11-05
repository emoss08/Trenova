package sessioncookie

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSOBindingCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &securityconfig.SessionConfig{Secure: true}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	token := SetSSOBinding(c, cfg)

	require.NotEmpty(t, token)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, ssoBindingSecureName, cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite, "the provider redirect is cross-site")
	assert.Equal(t, "/", cookies[0].Path)
	assert.Empty(t, cookies[0].Domain)

	back, _ := gin.CreateTestContext(httptest.NewRecorder())
	back.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	back.Request.AddCookie(cookies[0])

	assert.Equal(t, token, SSOBinding(back, cfg))
}

func TestSSOBindingIsEmptyWithoutTheCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	assert.Empty(t, SSOBinding(c, &securityconfig.SessionConfig{}))
}
