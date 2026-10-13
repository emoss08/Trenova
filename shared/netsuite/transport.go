package netsuite

import (
	"context"
	"net/http"
)

const preferHeader = "Prefer"

type requestHeadersKey struct{}

type requestHeaders struct {
	prefer string
}

func withRequestHeaders(ctx context.Context, headers requestHeaders) context.Context {
	if headers == (requestHeaders{}) {
		return ctx
	}
	return context.WithValue(ctx, requestHeadersKey{}, headers)
}

type headerTransport struct {
	next http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	headers, ok := req.Context().Value(requestHeadersKey{}).(requestHeaders)
	if !ok {
		return t.next.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	if headers.prefer != "" {
		clone.Header.Set(preferHeader, headers.prefer)
	}
	return t.next.RoundTrip(clone)
}

func headerClient(base *http.Client) *http.Client {
	var clone http.Client
	if base != nil {
		clone = *base
	}
	next := clone.Transport
	if next == nil {
		next = http.DefaultTransport
	}
	clone.Transport = &headerTransport{next: next}
	return &clone
}
