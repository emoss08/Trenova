package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func newTokenRedactionTestRouter(t *testing.T) (*gin.Engine, *observer.ObservedLogs) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	core, logs := observer.New(zapcore.InfoLevel)
	router := gin.New()
	router.Use(NewTokenRedactionMiddleware())
	router.Use(ginzap.Ginzap(zap.New(core), time.RFC3339, true))

	return router, logs
}

func observedLogText(t *testing.T, logs *observer.ObservedLogs) string {
	t.Helper()

	var sb []byte
	for _, entry := range logs.All() {
		sb = append(sb, entry.Message...)
		for _, field := range entry.Context {
			sb = fmt.Appendf(sb, " %s=%v", field.Key, field.Interface)
			sb = fmt.Appendf(sb, " %s=%s", field.Key, field.String)
		}
	}

	return string(sb)
}

func TestTokenRedactionMiddleware_RedactsPathParamToken(t *testing.T) {
	t.Parallel()

	const secret = "tok_01H00000000000000000000001"

	router, logs := newTokenRedactionTestRouter(t)

	var seenParam string
	var seenURL string
	router.GET("/tender-offers/:token/", func(c *gin.Context) {
		seenParam = c.Param("token")
		seenURL = c.Request.URL.String()
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tender-offers/"+secret+"/", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, secret, seenParam)
	assert.NotContains(t, seenURL, secret)
	assert.Contains(t, seenURL, redactedTokenPlaceholder)

	require.NotEmpty(t, logs.All())
	logText := observedLogText(t, logs)
	assert.NotContains(t, logText, secret)
	assert.Contains(t, logText, redactedTokenPlaceholder)
}

func TestTokenRedactionMiddleware_RedactsQueryParamToken(t *testing.T) {
	t.Parallel()

	const secret = "tok_01H00000000000000000000002"

	router, logs := newTokenRedactionTestRouter(t)

	var seenToken string
	var seenKeep string
	router.GET("/portal/invitations/preview", func(c *gin.Context) {
		seenToken = c.Query("token")
		seenKeep = c.Query("keep")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/portal/invitations/preview?token="+secret+"&keep=1",
		nil,
	)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, secret, seenToken)
	assert.Equal(t, "1", seenKeep)

	require.NotEmpty(t, logs.All())
	logText := observedLogText(t, logs)
	assert.NotContains(t, logText, secret)
	assert.Contains(t, logText, "token="+redactedTokenPlaceholder)
}

func TestTokenRedactionMiddleware_PassesThroughWithoutToken(t *testing.T) {
	t.Parallel()

	router, logs := newTokenRedactionTestRouter(t)

	const original = "/portal/invitations/preview?keep=1&other=abc"

	var seenURL string
	router.GET("/portal/invitations/preview", func(c *gin.Context) {
		seenURL = c.Request.URL.String()
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, original, nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, original, seenURL)

	require.NotEmpty(t, logs.All())
	logText := observedLogText(t, logs)
	assert.NotContains(t, logText, redactedTokenPlaceholder)
}

func TestTokenRedactionMiddleware_RedactsWebhookAndMailboxTokens(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		route  string
		secret string
	}{
		{
			name:   "webhook token",
			route:  "/webhooks/samsara/:webhookToken/",
			secret: "samsara_0123456789abcdef",
		},
		{
			name:   "mailbox token",
			route:  "/webhooks/inbound-mail/:mailboxToken/",
			secret: "Ab-_09mailboxSecret",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router, logs := newTokenRedactionTestRouter(t)
			param := strings.TrimSuffix(tc.route[strings.Index(tc.route, ":")+1:], "/")
			received := ""
			router.POST(tc.route, func(c *gin.Context) {
				received = c.Param(param)
				c.Status(http.StatusNoContent)
			})

			path := strings.Replace(tc.route, ":"+param, tc.secret, 1)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))

			assert.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, tc.secret, received, "the handler still reads the real token")
			logged := observedLogText(t, logs)
			assert.NotContains(t, logged, tc.secret)
			assert.Contains(t, logged, redactedTokenPlaceholder)
		})
	}
}

func TestTokenRedactionMiddleware_RedactsAnAS2InboundAddress(t *testing.T) {
	t.Parallel()

	const secret = "PROFILEADDRESS0000000000000000000000"

	router, logs := newTokenRedactionTestRouter(t)
	router.POST("/edi/as2/inbound/:as2InboundToken/", func(c *gin.Context) {
		assert.Equal(t, secret, c.Param("as2InboundToken"))
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/edi/as2/inbound/"+secret+"/", http.NoBody),
	)

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, observedLogText(t, logs), secret)
}
