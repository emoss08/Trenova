// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package middleware

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/pkg/metrics"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// DatabaseQueryHook provides monitoring and logging for database queries
type DatabaseQueryHook struct {
	log            *zerolog.Logger
	connectionName string
	enableMetrics  bool
}

// Ensure DatabaseQueryHook implements bun.QueryHook
var _ bun.QueryHook = (*DatabaseQueryHook)(nil)

// NewDatabaseQueryHook creates a new database query hook
func NewDatabaseQueryHook(
	log *zerolog.Logger,
	connectionName string,
	enableMetrics bool,
) *DatabaseQueryHook {
	return &DatabaseQueryHook{
		log:            log,
		connectionName: connectionName,
		enableMetrics:  enableMetrics,
	}
}

func (h *DatabaseQueryHook) BeforeQuery(
	ctx context.Context,
	event *bun.QueryEvent,
) context.Context {
	return ctx
}

func (h *DatabaseQueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	if !h.enableMetrics {
		return
	}

	duration := time.Since(event.StartTime)
	operation := h.getOperationType(event)

	// Record metrics
	metrics.RecordDatabaseOperation(operation, h.connectionName, duration)

	// Log slow queries
	if duration > 100*time.Millisecond {
		h.log.Warn().
			Str("connection", h.connectionName).
			Str("operation", operation).
			Dur("duration", duration).
			Str("query", event.Query).
			Msg("slow query detected")
	}
}

func (h *DatabaseQueryHook) getOperationType(event *bun.QueryEvent) string {
	// Extract operation type from query
	if event.QueryAppender != nil {
		switch event.QueryAppender.(type) {
		case *bun.SelectQuery:
			return "select"
		case *bun.InsertQuery:
			return "insert"
		case *bun.UpdateQuery:
			return "update"
		case *bun.DeleteQuery:
			return "delete"
		default:
			return "other"
		}
	}
	return "unknown"
}

// ConnectionTypeKey is used to store connection type in context
type connectionTypeKey struct{}

// WithConnectionType adds connection type to context
func WithConnectionType(ctx context.Context, connType string) context.Context {
	return context.WithValue(ctx, connectionTypeKey{}, connType)
}

// GetConnectionType retrieves connection type from context
func GetConnectionType(ctx context.Context) string {
	if connType, ok := ctx.Value(connectionTypeKey{}).(string); ok {
		return connType
	}
	return "unknown"
}
