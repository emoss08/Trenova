package base

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
)

func TraceURLBuilder(cfg *config.Config) func(traceID string) string {
	if cfg == nil {
		return func(string) string { return "" }
	}
	tracing := cfg.Monitoring.Tracing

	return tracing.TraceURL
}

func (r *Resolver) TraceURLOf(traceID string) *string {
	if r.TraceURL == nil || traceID == "" {
		return nil
	}

	url := r.TraceURL(traceID)
	if url == "" {
		return nil
	}

	return &url
}
