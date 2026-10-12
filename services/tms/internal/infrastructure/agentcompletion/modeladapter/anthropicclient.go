package modeladapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// anthropicBlockingTimeout bounds a blocking call whose HTTP client has no
// deadline. The SDK refuses a long blocking request without one.
const anthropicBlockingTimeout = 10 * time.Minute

// anthropicMessages builds the SDK's Messages service for one call from the
// provider alone. The SDK's default client reads ANTHROPIC_* credentials,
// base URL, profiles and headers from the process, which on a multi-tenant
// server would reach every provider an operator configured, so none of that
// is used. Retries stay with the router and the workflow, which count every
// attempt, and the HTTP client is the router's, whose dialer enforces the
// provider's egress policy.
func anthropicMessages(call *Call, stream bool) *anthropic.BetaMessageService {
	client := call.Client
	if stream {
		client = call.streamHTTPClient()
	}

	opts := []option.RequestOption{
		option.WithBaseURL(call.Provider.ResolvedBaseURL()),
		option.WithHTTPClient(client),
		option.WithMaxRetries(0),
		option.WithMiddleware(anthropicTransport(call, stream)),
	}
	if call.APIKey != "" {
		opts = append(opts, option.WithAPIKey(call.APIKey))
	}

	service := anthropic.NewBetaMessageService(opts...)

	return &service
}

func anthropicBlockingOptions(call *Call) []option.RequestOption {
	timeout := anthropicBlockingTimeout
	if call.Client != nil && call.Client.Timeout > 0 {
		timeout = call.Client.Timeout
	}

	return []option.RequestOption{option.WithRequestTimeout(timeout)}
}

// anthropicTransport turns a reply that is not 2xx into the TransportError
// every adapter returns, so retry, rest and failover read Anthropic the same
// way as before, and wraps a stream in the idle guard.
func anthropicTransport(call *Call, stream bool) option.Middleware {
	if stream {
		return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
			streamCtx, cancel := context.WithCancel(req.Context())
			resp, err := providerResponse(next(req.WithContext(streamCtx)))
			if err != nil {
				cancel()

				return nil, err
			}
			resp.Body = newIdleGuard(resp.Body, cancel, call.streamIdleTimeout())

			return resp, nil
		}
	}

	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		return providerResponse(next(req))
	}
}

func providerResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return nil, fmt.Errorf("execute provider request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		return nil, transportError(resp, payload)
	}

	return resp, nil
}
