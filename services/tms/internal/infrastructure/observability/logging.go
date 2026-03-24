package observability

import (
	"context"

	"go.uber.org/zap"
)

type ContextLogger struct {
	base *zap.Logger
}

func NewContextLogger(base *zap.Logger) *ContextLogger {
	return &ContextLogger{base: base}
}

func (l *ContextLogger) Logger() *zap.Logger {
	return l.base
}

func (l *ContextLogger) WithContext(ctx context.Context) *zap.Logger {
	fields := make([]zap.Field, 0, 5)

	if traceID := GetTraceID(ctx); traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID := GetSpanID(ctx); spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}
	if userID, ok := GetUserID(ctx); ok {
		fields = append(fields, zap.String("user_id", userID))
	}
	if orgID, ok := GetOrganizationID(ctx); ok {
		fields = append(fields, zap.String("org_id", orgID))
	}
	if requestID, ok := GetRequestID(ctx); ok {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if len(fields) == 0 {
		return l.base
	}

	return l.base.With(fields...)
}

func (l *ContextLogger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Debug(msg, fields...)
}

func (l *ContextLogger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Info(msg, fields...)
}

func (l *ContextLogger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Warn(msg, fields...)
}

func (l *ContextLogger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).Error(msg, fields...)
}

func (l *ContextLogger) With(fields ...zap.Field) *ContextLogger {
	return &ContextLogger{base: l.base.With(fields...)}
}

func (l *ContextLogger) Named(name string) *ContextLogger {
	return &ContextLogger{base: l.base.Named(name)}
}
