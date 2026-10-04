package turnstile

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type siteverify struct {
	status int
	body   string
	delay  time.Duration
	form   atomic.Pointer[url.Values]
	calls  atomic.Int32
}

func (s *siteverify) server(t *testing.T) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls.Add(1)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		if err := r.ParseForm(); err == nil {
			form := r.PostForm
			s.form.Store(&form)
		}
		if s.delay > 0 {
			time.Sleep(s.delay)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(s.body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func newTestVerifier(t *testing.T, sv *siteverify, secret string) *Verifier {
	t.Helper()

	srv := sv.server(t)
	return NewVerifier(&Options{
		Enabled:          true,
		SecretKey:        secret,
		VerifyURL:        srv.URL,
		Timeout:          time.Second,
		ExpectedHostname: "app.trenova.test",
	})
}

func TestVerifyAcceptsAMatchingToken(t *testing.T) {
	t.Parallel()

	sv := &siteverify{
		status: http.StatusOK,
		body:   `{"success":true,"hostname":"app.trenova.test","action":"signup","error-codes":[]}`,
	}
	v := newTestVerifier(t, sv, "real-secret")

	err := v.Verify(t.Context(), &services.TurnstileVerification{
		Token:          "token-1",
		RemoteIP:       "203.0.113.9",
		ExpectedAction: services.TurnstileActionSignup,
		IdempotencyKey: "idem-1",
	})
	require.NoError(t, err)

	form := sv.form.Load()
	require.NotNil(t, form)
	assert.Equal(t, "real-secret", form.Get("secret"))
	assert.Equal(t, "token-1", form.Get("response"))
	assert.Equal(t, "203.0.113.9", form.Get("remoteip"))
	assert.Equal(t, "idem-1", form.Get("idempotency_key"))
}

func TestVerifyRejections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		body   string
		action string
		token  string
	}{
		{
			name:   "unsuccessful",
			body:   `{"success":false,"error-codes":["timeout-or-duplicate"]}`,
			action: services.TurnstileActionSignup,
			token:  "t",
		},
		{
			name:   "wrong hostname",
			body:   `{"success":true,"hostname":"evil.example","action":"signup"}`,
			action: services.TurnstileActionSignup,
			token:  "t",
		},
		{
			name:   "wrong action",
			body:   `{"success":true,"hostname":"app.trenova.test","action":"signup"}`,
			action: services.TurnstileActionSignupResend,
			token:  "t",
		},
		{
			name:   "unknown expected action",
			body:   `{"success":true,"hostname":"app.trenova.test","action":"login"}`,
			action: "login",
			token:  "t",
		},
		{
			name:   "empty token",
			body:   `{"success":true}`,
			action: services.TurnstileActionSignup,
			token:  "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sv := &siteverify{status: http.StatusOK, body: tt.body}
			v := newTestVerifier(t, sv, "real-secret")

			err := v.Verify(t.Context(), &services.TurnstileVerification{
				Token:          tt.token,
				ExpectedAction: tt.action,
			})
			require.ErrorIs(t, err, services.ErrTurnstileRejected)
		})
	}
}

func TestVerifyEmptyTokenNeverCallsCloudflare(t *testing.T) {
	t.Parallel()

	sv := &siteverify{status: http.StatusOK, body: `{"success":true}`}
	v := newTestVerifier(t, sv, "real-secret")

	err := v.Verify(t.Context(), &services.TurnstileVerification{Token: ""})
	require.ErrorIs(t, err, services.ErrTurnstileRejected)
	assert.Zero(t, sv.calls.Load())
}

func TestVerifyUnavailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		body   string
		delay  time.Duration
	}{
		{name: "server error", status: http.StatusInternalServerError, body: `oops`},
		{name: "malformed body", status: http.StatusOK, body: `{not json`},
		{name: "timeout", status: http.StatusOK, body: `{"success":true}`, delay: 300 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sv := &siteverify{status: tt.status, body: tt.body, delay: tt.delay}
			srv := sv.server(t)
			v := NewVerifier(&Options{
				Enabled:   true,
				SecretKey: "real-secret",
				VerifyURL: srv.URL,
				Timeout:   100 * time.Millisecond,
			})

			err := v.Verify(t.Context(), &services.TurnstileVerification{
				Token:          "t",
				ExpectedAction: services.TurnstileActionSignup,
			})
			require.ErrorIs(t, err, services.ErrTurnstileUnavailable)
		})
	}
}

func TestTestKeysSkipHostnameAndEmptyAction(t *testing.T) {
	t.Parallel()

	sv := &siteverify{
		status: http.StatusOK,
		body:   `{"success":true,"hostname":"example.com","action":""}`,
	}
	v := newTestVerifier(t, sv, "1x0000000000000000000000000000000AA")

	err := v.Verify(t.Context(), &services.TurnstileVerification{
		Token:          "XXXX.DUMMY.TOKEN.XXXX",
		ExpectedAction: services.TurnstileActionSignupResend,
	})
	require.NoError(t, err)
}

func TestDisabledVerifierAcceptsEverything(t *testing.T) {
	t.Parallel()

	v := NewVerifier(&Options{Enabled: false})
	assert.False(t, v.Enabled())
	require.NoError(t, v.Verify(t.Context(), &services.TurnstileVerification{}))
}

func TestHostnameOf(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "app.trenova.app", hostnameOf("https://app.trenova.app/"))
	assert.Equal(t, "localhost", hostnameOf("http://localhost:5173"))
	assert.Empty(t, hostnameOf(""))
}
