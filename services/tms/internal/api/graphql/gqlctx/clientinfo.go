package gqlctx

import "context"

type clientInfoKey struct{}

// ClientInfo is where a request came from: the address and the client. It is
// carried for the few operations that record it as evidence — an electronic
// signature is a name, a moment, and where the moment happened — and nothing
// else reads it.
type ClientInfo struct {
	IP        string
	UserAgent string
}

func WithClientInfo(ctx context.Context, info ClientInfo) context.Context {
	return context.WithValue(ctx, clientInfoKey{}, info)
}

// ClientInfoFrom is the request's origin, or an empty one when the context
// did not come through the HTTP handler.
func ClientInfoFrom(ctx context.Context) ClientInfo {
	info, _ := ctx.Value(clientInfoKey{}).(ClientInfo)
	return info
}
