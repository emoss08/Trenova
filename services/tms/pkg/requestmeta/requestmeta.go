package requestmeta

import (
	"context"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	GinContextKey      = "trenova.requestmeta"
	maxUserAgentLength = 255
	maxClientIPLength  = 45
)

type Meta struct {
	RequestID string
	ClientIP  string
	UserAgent string
}

func New(requestID, clientIP, userAgent string) Meta {
	return Meta{
		RequestID: requestID,
		ClientIP:  stringutils.TruncateRunes(clientIP, maxClientIPLength),
		UserAgent: stringutils.TruncateRunes(userAgent, maxUserAgentLength),
	}
}

type metaKey struct{}

func With(ctx context.Context, meta Meta) context.Context {
	return context.WithValue(ctx, metaKey{}, meta)
}

func From(ctx context.Context) (Meta, bool) {
	if ctx == nil {
		return Meta{}, false
	}

	if meta, ok := ctx.Value(metaKey{}).(Meta); ok {
		return meta, true
	}

	meta, ok := ctx.Value(GinContextKey).(Meta)

	return meta, ok
}
