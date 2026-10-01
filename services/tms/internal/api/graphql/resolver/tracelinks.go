package resolver

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
)

func traceURLBuilder(cfg *config.Config) func(traceID string) string {
	if cfg == nil {
		return func(string) string { return "" }
	}
	tracing := cfg.Monitoring.Tracing

	return tracing.TraceURL
}

func (r *Resolver) traceURLOf(traceID string) *string {
	if r.traceURL == nil || traceID == "" {
		return nil
	}

	url := r.traceURL(traceID)
	if url == "" {
		return nil
	}

	return &url
}
