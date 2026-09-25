package config

import (
	"errors"
	"net/url"
	"strings"
)

const (
	defaultAISamplingRate      = 1.0
	TraceURLTraceIDPlaceholder = "{traceId}"
	traceURLProbeTraceID       = "4bf92f3577b34da6a3ce929d0e0e4736"
)

var (
	ErrTraceURLTemplateMissingPlaceholder = errors.New(
		"monitoring.tracing.traceUrlTemplate must contain " + TraceURLTraceIDPlaceholder,
	)
	ErrTraceURLTemplateInvalid = errors.New(
		"monitoring.tracing.traceUrlTemplate must be an absolute http or https URL",
	)
)

func (c *TracingConfig) GetAISamplingRate() float64 {
	if c.AISamplingRate == nil {
		return defaultAISamplingRate
	}

	return *c.AISamplingRate
}

func (c *TracingConfig) TraceURL(traceID string) string {
	template := strings.TrimSpace(c.TraceURLTemplate)
	if template == "" || traceID == "" {
		return ""
	}

	return strings.ReplaceAll(template, TraceURLTraceIDPlaceholder, url.PathEscape(traceID))
}

func validateTracingConfig(config *Config) error {
	template := strings.TrimSpace(config.Monitoring.Tracing.TraceURLTemplate)
	if template == "" {
		return nil
	}

	if !strings.Contains(template, TraceURLTraceIDPlaceholder) {
		return ErrTraceURLTemplateMissingPlaceholder
	}

	probe, err := url.Parse(
		strings.ReplaceAll(template, TraceURLTraceIDPlaceholder, traceURLProbeTraceID),
	)
	if err != nil || probe.Host == "" || (probe.Scheme != "http" && probe.Scheme != "https") {
		return ErrTraceURLTemplateInvalid
	}

	return nil
}
