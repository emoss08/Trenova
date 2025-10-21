package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// OrganizationIDKey is the context key for organization ID
	OrganizationIDKey ContextKey = "organization_id"
	// APIKeyIDKey is the context key for API key ID
	APIKeyIDKey ContextKey = "api_key_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
)

// StartSpanFromContext starts a new span from the given context.
// The returned span must be ended by calling span.End().
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

func WithUserID(ctx context.Context, userID string) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, userID)
	AddSpanAttributes(ctx, attribute.String("user.id", userID))

	member, _ := baggage.NewMember("user.id", userID)
	bag, _ := baggage.New(member)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return ctx
}

func WithOrganizationID(ctx context.Context, orgID string) context.Context {
	ctx = context.WithValue(ctx, OrganizationIDKey, orgID)
	AddSpanAttributes(ctx, attribute.String("organization.id", orgID))

	member, _ := baggage.NewMember("organization.id", orgID)
	bag, _ := baggage.New(member)
	ctx = baggage.ContextWithBaggage(ctx, bag)

	return ctx
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

func FormatTraceURL(traceID, provider, endpoint string) string {
	switch provider {
	case "jaeger":
		return fmt.Sprintf("%s/trace/%s", endpoint, traceID)
	case "zipkin":
		return fmt.Sprintf("%s/zipkin/traces/%s", endpoint, traceID)
	default:
		return fmt.Sprintf("trace://%s", traceID)
	}
}
