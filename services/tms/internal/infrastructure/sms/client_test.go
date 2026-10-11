package sms

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testConfig() *config.TwilioConfig {
	return &config.TwilioConfig{
		Enabled:    true,
		AccountSID: "AC123",
		AuthToken:  "secret",
		FromNumber: "+15550000000",
	}
}

func newTestClient(t *testing.T, cfg *config.TwilioConfig, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return newClient(cfg, zap.NewNop(), server.Client(), server.URL)
}

func TestSendPostsMessageForm(t *testing.T) {
	t.Parallel()

	var (
		gotPath   string
		gotForm   url.Values
		gotUser   string
		gotPass   string
		gotAuthOK bool
		gotType   string
	)
	client := newTestClient(t, testConfig(), func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotType = r.Header.Get("Content-Type")
		gotUser, gotPass, gotAuthOK = r.BasicAuth()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotForm, err = url.ParseQuery(string(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM1","status":"queued"}`))
	})

	err := client.Send(t.Context(), SendRequest{To: "+15551234567", Body: "Load 42 delivered"})

	require.NoError(t, err)
	assert.Equal(t, "/Accounts/AC123/Messages.json", gotPath)
	assert.Equal(t, "application/x-www-form-urlencoded", gotType)
	assert.True(t, gotAuthOK)
	assert.Equal(t, "AC123", gotUser)
	assert.Equal(t, "secret", gotPass)
	assert.Equal(t, "+15551234567", gotForm.Get("To"))
	assert.Equal(t, "+15550000000", gotForm.Get("From"))
	assert.Equal(t, "Load 42 delivered", gotForm.Get("Body"))
}

func TestSendEscapesAccountSID(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.AccountSID = "AC/../x"
	var gotPath string
	client := newTestClient(t, cfg, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM1"}`))
	})

	require.NoError(t, client.Send(t.Context(), SendRequest{To: "+1", Body: "x"}))
	assert.Equal(t, "/Accounts/AC%2F..%2Fx/Messages.json", gotPath)
}

func TestSendReturnsPermanentAPIError(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, testConfig(), func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(
			`{"code":21211,"message":"The 'To' number is not a valid phone number.",` +
				`"more_info":"https://www.twilio.com/docs/errors/21211","status":400}`,
		))
	})

	err := client.Send(t.Context(), SendRequest{To: "bad", Body: "x"})

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, 21211, apiErr.Code)
	assert.Equal(t, "The 'To' number is not a valid phone number.", apiErr.Message)
	assert.Equal(t, "https://www.twilio.com/docs/errors/21211", apiErr.MoreInfo)
	assert.False(t, apiErr.Retryable())
}

func TestSendReturnsRetryableAPIErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		retryAfter string
		body       string
		wantDelay  int
		wantMsg    string
	}{
		{
			name:       "rate limited with retry-after",
			status:     http.StatusTooManyRequests,
			retryAfter: "12",
			body:       `{"code":20429,"message":"Too Many Requests"}`,
			wantDelay:  12,
			wantMsg:    "Too Many Requests",
		},
		{
			name:    "server error without a json body",
			status:  http.StatusServiceUnavailable,
			body:    "upstream unavailable",
			wantMsg: http.StatusText(http.StatusServiceUnavailable),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := newTestClient(t, testConfig(), func(w http.ResponseWriter, _ *http.Request) {
				if tt.retryAfter != "" {
					w.Header().Set("Retry-After", tt.retryAfter)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			err := client.Send(t.Context(), SendRequest{To: "+1", Body: "x"})

			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.True(t, apiErr.Retryable())
			assert.Equal(t, tt.status, apiErr.StatusCode)
			assert.Equal(t, tt.wantDelay, apiErr.RetryAfter)
			assert.Equal(t, tt.wantMsg, apiErr.Message)
		})
	}
}

func TestSendRefusesWhenDisabled(t *testing.T) {
	t.Parallel()

	cfg := testConfig()
	cfg.Enabled = false
	called := false
	client := newTestClient(t, cfg, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	err := client.Send(t.Context(), SendRequest{To: "+1", Body: "x"})

	require.ErrorIs(t, err, ErrNotConfigured)
	assert.False(t, called)
}

func TestSendHonoursContextCancellation(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	client := newTestClient(t, testConfig(), func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusCreated)
	})
	t.Cleanup(func() { close(release) })

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	err := client.Send(ctx, SendRequest{To: "+1", Body: "x"})

	require.ErrorIs(t, err, context.DeadlineExceeded)
}
