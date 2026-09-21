package aiproviderservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeTimeoutError struct{}

func (fakeTimeoutError) Error() string   { return "i/o timeout" }
func (fakeTimeoutError) Timeout() bool   { return true }
func (fakeTimeoutError) Temporary() bool { return true }

func TestIsTimeout_ReadsTheShapesATransportActuallyReturns(t *testing.T) {
	// http2's own wording is a plain error, not a net.Error, and it is what
	// a queued endpoint produced in the field.
	http2Style := errors.New(
		`Post "https://integrate.api.nvidia.com/v1/chat/completions": ` +
			`http2: timeout awaiting response headers ` +
			`(Client.Timeout exceeded while awaiting headers)`,
	)

	cases := map[string]struct {
		err  error
		want bool
	}{
		"http2 awaiting headers": {err: http2Style, want: true},
		"context deadline":       {err: context.DeadlineExceeded, want: true},
		"wrapped context deadline": {
			err:  fmt.Errorf("execute provider request: %w", context.DeadlineExceeded),
			want: true,
		},
		"net.Error that times out": {
			err: &url.Error{
				Op:  "Post",
				URL: "https://example.test/v1/chat/completions",
				Err: fakeTimeoutError{},
			},
			want: true,
		},
		"connection refused": {
			err: &url.Error{
				Op:  "Post",
				URL: "https://example.test/v1/chat/completions",
				Err: &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			},
			want: false,
		},
		"dns failure": {err: errors.New("no such host"), want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, isTimeout(tc.err))
		})
	}
}

func TestUnreachable_SeparatesABusyEndpointFromAMissingOne(t *testing.T) {
	prober := &Prober{
		logger: zap.NewNop(),
		cfg:    &config.AIConfig{},
	}

	timedOut := prober.unreachable(
		errors.New("http2: timeout awaiting response headers"),
		20_000,
	)
	require.False(t, timedOut.Success)
	assert.Equal(t, "The endpoint did not answer in time", timedOut.Message)
	// The remedy names the setting rather than sending somebody to re-check
	// a base URL that was right all along.
	assert.Contains(t, timedOut.Detail, "ai.probeTimeout")
	assert.Contains(t, timedOut.Detail, "45s")

	refused := prober.unreachable(errors.New("connection refused"), 12)
	require.False(t, refused.Success)
	assert.Equal(t, "Could not reach the endpoint", refused.Message)
	assert.Equal(t, "connection refused", refused.Detail)
}

// serverRequestTimeout is what config.example.yaml ships for
// server.requestTimeout. A probe that outlives it returns a 504 instead of
// a verdict, which is the failure this whole change is about.
const serverRequestTimeout = 55 * time.Second

func TestGetProbeTimeout_OutlastsAQueuedFreeTier(t *testing.T) {
	cfg := &config.AIConfig{}
	// The probe waits on a real generation, so it must not inherit the
	// reachability budget that failed NVIDIA's queued endpoints at twenty
	// seconds. It must still fit inside the server's own request timeout.
	assert.Greater(t, cfg.GetProbeTimeout(), cfg.GetTimeout())
	assert.Less(t, cfg.GetProbeTimeout(), serverRequestTimeout)
}

func TestEmptyReplyAdvice_NamesTheReasoningBudgetRatherThanTheEndpoint(t *testing.T) {
	// A Nemotron-class model on a free tier thinks first. With a small
	// ceiling the whole budget goes on the chain of thought and the reply
	// arrives empty, which is a limit to raise, not an endpoint to replace.
	thinking := emptyReplyAdvice(&modeladapter.Response{ReasoningTokens: 240})
	assert.Contains(t, thinking, "240 tokens thinking")
	assert.Contains(t, thinking, "max tokens")

	truncated := emptyReplyAdvice(&modeladapter.Response{Truncated: true})
	assert.Contains(t, truncated, "output limit")

	silent := emptyReplyAdvice(&modeladapter.Response{})
	assert.Contains(t, silent, "model name")
}
