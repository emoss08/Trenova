package gqlctx

import (
	"context"

	"github.com/emoss08/trenova/pkg/authctx"
)

type authContextKey struct{}
type requestIDKey struct{}

func WithAuthContext(ctx context.Context, auth *authctx.AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey{}, auth)
}

func AuthContext(ctx context.Context) (*authctx.AuthContext, bool) {
	auth, ok := ctx.Value(authContextKey{}).(*authctx.AuthContext)
	return auth, ok
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}
