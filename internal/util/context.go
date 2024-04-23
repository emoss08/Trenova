package util

import (
	"context"
)

type ContextKey string

const (
	CTXKeyDisableLogger = ContextKey("disableLogger")
	CTXOrganizationID   = ContextKey("organizationID")
	CTXBusinessUnitID   = ContextKey("businessUnitID")
	CTXUserID           = ContextKey("userID")
)

// ShouldDisableLogger checks whether the logger instance should be disabled for the provided context.
// `util.LogFromContext` will use this function to check whether it should return a default logger if
// none has been set by our logging middleware before, or fall back to the disabled logger, suppressing
// all output. Use `ctx = util.DisableLogger(ctx, true)` to disable logging for the given context.
func ShouldDisableLogger(ctx context.Context) bool {
	s := ctx.Value(CTXKeyDisableLogger)
	if s == nil {
		return false
	}

	shouldDisable, ok := s.(bool)
	if !ok {
		return false
	}

	return shouldDisable
}

// DisableLogger toggles the indication whether `util.LogFromContext` should return a disabled logger
// for a context if none has been set by our logging middleware before. Whilst the usecase for a disabled
// logger are relatively minimal (we almost always want to have some log output, even if the context
// was not directly derived from a HTTP request), this functionality was provideds so you can switch back
// to the old zerolog behavior if so desired.
func DisableLogger(ctx context.Context, shouldDisable bool) context.Context {
	return context.WithValue(ctx, CTXKeyDisableLogger, shouldDisable)
}
