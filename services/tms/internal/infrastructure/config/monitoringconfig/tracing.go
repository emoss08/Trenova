package monitoringconfig

import (
	"errors"
	"strings"

	"github.com/emoss08/trenova/shared/urlutils"
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

func (c *TracingConfig) Validate() error {
	template := strings.TrimSpace(c.TraceURLTemplate)
	if template == "" {
		return nil
	}

	if !strings.Contains(template, TraceURLTraceIDPlaceholder) {
		return ErrTraceURLTemplateMissingPlaceholder
	}

	if _, ok := urlutils.ParseAbsoluteHTTP(
		strings.ReplaceAll(template, TraceURLTraceIDPlaceholder, traceURLProbeTraceID),
	); !ok {
		return ErrTraceURLTemplateInvalid
	}

	return nil
}
