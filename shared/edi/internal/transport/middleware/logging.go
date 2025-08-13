package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/transport"
	"github.com/go-kit/kit/endpoint"
	"go.uber.org/zap"
)

// LoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func LoggingMiddleware(logger *zap.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				fields := []zap.Field{
					zap.Duration("duration", time.Since(begin)),
					zap.String("request_type", fmt.Sprintf("%T", request)),
				}

				// Add request ID if available
				if reqID := ctx.Value("request_id"); reqID != nil {
					fields = append(fields, zap.String("request_id", reqID.(string)))
				}

				// Log based on error presence
				if err != nil {
					// Determine log level based on error type
					var serviceErr transport.ServiceError
					switch {
					case serviceErr.Type == transport.ErrorTypeInternal:
						logger.Error("endpoint error", append(fields, zap.Error(err))...)
					case serviceErr.Type == transport.ErrorTypeValidation:
						logger.Warn("validation error", append(fields, zap.Error(err))...)
					default:
						logger.Info("endpoint completed with error", append(fields, zap.Error(err))...)
					}
				} else {
					// Check if response contains an error field
					if resp, ok := response.(interface{ GetError() string }); ok && resp.GetError() != "" {
						logger.Warn("endpoint completed with business error",
							append(fields, zap.String("error", resp.GetError()))...)
					} else {
						logger.Info("endpoint completed successfully", fields...)
					}
				}
			}(time.Now())

			return next(ctx, request)
		}
	}
}

// DetailedLoggingMiddleware provides more detailed logging including request/response bodies
func DetailedLoggingMiddleware(logger *zap.Logger, logPayloads bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			start := time.Now()
			
			// Log request
			fields := []zap.Field{
				zap.String("request_type", fmt.Sprintf("%T", request)),
				zap.Time("started_at", start),
			}

			if reqID := ctx.Value("request_id"); reqID != nil {
				fields = append(fields, zap.String("request_id", reqID.(string)))
			}

			if logPayloads {
				fields = append(fields, zap.Any("request_payload", request))
			}

			logger.Debug("endpoint request started", fields...)

			// Execute endpoint
			response, err = next(ctx, request)

			// Log response
			duration := time.Since(start)
			responseFields := []zap.Field{
				zap.Duration("duration", duration),
				zap.String("request_type", fmt.Sprintf("%T", request)),
			}

			if reqID := ctx.Value("request_id"); reqID != nil {
				responseFields = append(responseFields, zap.String("request_id", reqID.(string)))
			}

			if err != nil {
				responseFields = append(responseFields, zap.Error(err))
				logger.Error("endpoint request failed", responseFields...)
			} else {
				if logPayloads && response != nil {
					responseFields = append(responseFields, zap.Any("response_payload", response))
				}
				
				// Warn on slow requests
				if duration > 5*time.Second {
					logger.Warn("slow endpoint request", responseFields...)
				} else {
					logger.Debug("endpoint request completed", responseFields...)
				}
			}

			return response, err
		}
	}
}

// ContextLogger adds logger to context for use in services
func ContextLogger(logger *zap.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			// Add logger to context with request-specific fields
			if reqID := ctx.Value("request_id"); reqID != nil {
				ctx = context.WithValue(ctx, "logger", logger.With(zap.String("request_id", reqID.(string))))
			} else {
				ctx = context.WithValue(ctx, "logger", logger)
			}
			return next(ctx, request)
		}
	}
}

// GetLoggerFromContext retrieves the logger from context
func GetLoggerFromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value("logger").(*zap.Logger); ok {
		return logger
	}
	// Return a default logger if none in context
	logger, _ := zap.NewProduction()
	return logger
}