package i18n

import (
	"context"
	"net/http"
)

type contextKey struct{}

func WithLocale(ctx context.Context, locale Locale) context.Context {
	if !locale.IsValid() {
		locale = Default
	}
	return context.WithValue(ctx, contextKey{}, locale)
}

func FromContext(ctx context.Context) Locale {
	if ctx == nil {
		return Default
	}
	if locale, ok := ctx.Value(contextKey{}).(Locale); ok && locale.IsValid() {
		return locale
	}
	return Default
}

func FromRequest(r *http.Request) Locale {
	if r == nil {
		return Default
	}

	if locale, ok := r.Context().Value(contextKey{}).(Locale); ok && locale.IsValid() {
		return locale
	}

	return ParseAcceptLanguage(r.Header.Get("Accept-Language"))
}
