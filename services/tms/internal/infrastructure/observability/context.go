package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ContextKey string

const (
	TraceIDKey        = ContextKey("trace_id")
	SpanIDKey         = ContextKey("span_id")
	UserIDKey         = ContextKey("user_id")
	OrganizationIDKey = ContextKey("organization_id")
	APIKeyIDKey       = ContextKey("api_key_id")
	RequestIDKey      = ContextKey("request_id")
)

func StartSpanFromContext(
	ctx context.Context,
	name string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	ctx, span := otel.Tracer("trenova"). //nolint:spancheck // This will be checked by the caller
						Start(ctx, name, opts...)
	return ctx, span //nolint:spancheck // This will be checked by the caller
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

func RecordSpanError(ctx context.Context, err error, opts ...trace.EventOption) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err, opts...)
		span.SetStatus(codes.Error, err.Error())
	}
}

func SetSpanOK(ctx context.Context, message string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(codes.Ok, message)
	}
}

func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}

	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}

	return ""
}

func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}

	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}

	return ""
}

func WithTraceIDs(ctx context.Context) context.Context {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		ctx = context.WithValue(ctx, TraceIDKey, span.SpanContext().TraceID().String())
	}
	if span.SpanContext().HasSpanID() {
		ctx = context.WithValue(ctx, SpanIDKey, span.SpanContext().SpanID().String())
	}
	return ctx
}

func withBaggageMember(ctx context.Context, key, value string) context.Context {
	member, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx
	}

	bag := baggage.FromContext(ctx)
	bag, err = bag.SetMember(member)
	if err != nil {
		return ctx
	}

	return baggage.ContextWithBaggage(ctx, bag)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	AddSpanAttributes(ctx, attribute.String("user.id", userID))
	return withBaggageMember(ctx, "user.id", userID)
}

func WithOrganizationID(ctx context.Context, orgID string) context.Context {
	ctx = context.WithValue(ctx, OrganizationIDKey, orgID)
	AddSpanAttributes(ctx, attribute.String("organization.id", orgID))
	return withBaggageMember(ctx, "organization.id", orgID)
}

func WithAPIKeyID(ctx context.Context, apiKeyID string) context.Context {
	ctx = context.WithValue(ctx, APIKeyIDKey, apiKeyID)
	AddSpanAttributes(ctx, attribute.String("api.key_id", apiKeyID))
	return ctx
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, RequestIDKey, requestID)
	AddSpanAttributes(ctx, attribute.String("request.id", requestID))
	return ctx
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		bag := baggage.FromContext(ctx)
		member := bag.Member("user.id")
		if member.Value() != "" {
			return member.Value(), true
		}
	}
	return userID, ok
}

func GetOrganizationID(ctx context.Context) (string, bool) {
	orgID, ok := ctx.Value(OrganizationIDKey).(string)
	if !ok {
		bag := baggage.FromContext(ctx)
		member := bag.Member("organization.id")
		if member.Value() != "" {
			return member.Value(), true
		}
	}
	return orgID, ok
}

func GetAPIKeyID(ctx context.Context) (string, bool) {
	apiKeyID, ok := ctx.Value(APIKeyIDKey).(string)
	return apiKeyID, ok
}

func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

func RunWithSpan(
	ctx context.Context,
	name string,
	fn func(context.Context) error,
	opts ...trace.SpanStartOption,
) error {
	ctx, span := StartSpanFromContext(ctx, name, opts...)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		RecordSpanError(ctx, err)
		return err
	}

	SetSpanOK(ctx, "completed successfully")
	return nil
}

func RunWithSpanReturn[T any](
	ctx context.Context,
	name string,
	fn func(context.Context) (T, error),
	opts ...trace.SpanStartOption,
) (T, error) {
	ctx, span := StartSpanFromContext(ctx, name, opts...)
	defer span.End()

	result, err := fn(ctx)
	if err != nil {
		RecordSpanError(ctx, err)
		return result, err
	}

	SetSpanOK(ctx, "completed successfully")
	return result, nil
}

func ExtractTraceState(ctx context.Context) map[string]string {
	state := make(map[string]string)

	if traceID := GetTraceID(ctx); traceID != "" {
		state["trace_id"] = traceID
	}

	if spanID := GetSpanID(ctx); spanID != "" {
		state["span_id"] = spanID
	}

	if userID, ok := GetUserID(ctx); ok {
		state["user_id"] = userID
	}

	if orgID, ok := GetOrganizationID(ctx); ok {
		state["organization_id"] = orgID
	}

	if requestID, ok := GetRequestID(ctx); ok {
		state["request_id"] = requestID
	}

	return state
}
