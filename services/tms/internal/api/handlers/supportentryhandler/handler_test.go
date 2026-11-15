package supportentryhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type entrance struct {
	calls int
	got   string
	err   error
}

func (e *entrance) Enter(
	_ context.Context,
	req *supportaccessservice.EnterRequest,
) (*supportaccessservice.EnterResult, error) {
	e.calls++
	e.got = req.Assertion
	if e.err != nil {
		return nil, e.err
	}
	return &supportaccessservice.EnterResult{Token: "ses_x.secret", ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
}

type limiter struct {
	allow bool
	err   error
}

func (l *limiter) Check(
	_ context.Context,
	reqs []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	if l.err != nil {
		return nil, l.err
	}
	out := make([]kernelports.RateLimitDecision, len(reqs))
	for i := range out {
		out[i] = kernelports.RateLimitDecision{Allowed: l.allow, RetryAfter: 30 * time.Second}
	}
	return out, nil
}

func newRouter(e Entrance, l kernelports.RateLimitStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &Handler{
		entrance: e,
		limits:   l,
		managed:  true,
		session: &securityconfig.SessionConfig{
			Name: "trenova_session", Path: "/", Secure: true, HTTPOnly: true, SameSite: "strict",
		},
		l: zap.NewNop(),
	}
	router := gin.New()
	h.RegisterPublicRoutes(router.Group("/api/v1"))
	return router
}

func post(router *gin.Engine, target, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func form(assertion string) string {
	return url.Values{"assertion": {assertion}}.Encode()
}

func TestEnter_OpensTheSessionAndSendsTheBrowserIntoTheApp(t *testing.T) {
	t.Parallel()

	e := &entrance{}
	rec := post(newRouter(e, &limiter{allow: true}), "/api/v1/support/enter",
		"application/x-www-form-urlencoded", form("header.payload.signature"))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "header.payload.signature", e.got)
	assert.Contains(t, rec.Body.String(), `http-equiv="refresh" content="0;url=/"`)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	cookies := rec.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "trenova_session", cookies[0].Name)
	assert.True(t, cookies[0].HttpOnly)
	assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
}

func TestEnter_AnswersEveryRefusalTheSameWay(t *testing.T) {
	t.Parallel()

	var bodies []string
	for _, reason := range []supportaccess.RefusalReason{
		supportaccess.RefusalNoGrant,
		supportaccess.RefusalExpired,
		supportaccess.RefusalReplayed,
		supportaccess.RefusalSignature,
		supportaccess.RefusalAudience,
	} {
		e := &entrance{err: &supportaccess.RefusalError{Reason: reason}}
		rec := post(newRouter(e, &limiter{allow: true}), "/api/v1/support/enter",
			"application/x-www-form-urlencoded", form("a.b.c"))
		assert.Equal(t, http.StatusForbidden, rec.Code, reason)
		assert.Empty(t, rec.Result().Cookies(), reason)
		assert.NotContains(t, rec.Body.String(), string(reason))
		bodies = append(bodies, rec.Body.String())
	}
	for _, body := range bodies {
		assert.Equal(t, bodies[0], body, "a refusal never says why")
	}
}

func TestEnter_RefusesAnAssertionAnywhereButAFormBody(t *testing.T) {
	t.Parallel()

	e := &entrance{}
	router := newRouter(e, &limiter{allow: true})

	rec := post(router, "/api/v1/support/enter?assertion=a.b.c", "application/x-www-form-urlencoded", form("a.b.c"))
	assert.Equal(t, http.StatusForbidden, rec.Code, "an assertion in a URL is logged and kept in history")

	rec = post(router, "/api/v1/support/enter", "application/json", `{"assertion":"a.b.c"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = post(router, "/api/v1/support/enter", "application/x-www-form-urlencoded",
		url.Values{"assertion": {"a.b.c", "d.e.f"}}.Encode())
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = post(router, "/api/v1/support/enter", "application/x-www-form-urlencoded",
		url.Values{"assertion": {"a.b.c"}, "next": {"/admin"}}.Encode())
	assert.Equal(t, http.StatusForbidden, rec.Code)

	assert.Zero(t, e.calls)
}

func TestEnter_RefusesAnOversizedBody(t *testing.T) {
	t.Parallel()

	e := &entrance{}
	rec := post(newRouter(e, &limiter{allow: true}), "/api/v1/support/enter",
		"application/x-www-form-urlencoded", form(strings.Repeat("a", maxBody+1)))

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Zero(t, e.calls)
}

func TestEnter_RefusesOverItsRateLimitAndWhenTheLimiterIsDown(t *testing.T) {
	t.Parallel()

	e := &entrance{}
	rec := post(newRouter(e, &limiter{allow: false}), "/api/v1/support/enter",
		"application/x-www-form-urlencoded", form("a.b.c"))
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "30", rec.Header().Get("Retry-After"))

	rec = post(newRouter(e, &limiter{err: errors.New("redis down")}), "/api/v1/support/enter",
		"application/x-www-form-urlencoded", form("a.b.c"))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	assert.Zero(t, e.calls, "nothing is verified past the limit")
}

func TestEnter_IsNotRoutedOnASelfHostedInstall(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	h := &Handler{entrance: &entrance{}, limits: &limiter{allow: true}, managed: false, l: zap.NewNop()}
	router := gin.New()
	h.RegisterPublicRoutes(router.Group("/api/v1"))

	rec := post(router, "/api/v1/support/enter", "application/x-www-form-urlencoded", form("a.b.c"))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
