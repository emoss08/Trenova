//revive:disable-next-line:var-naming
package metrics

import (
	"fmt"
	"sync"

	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type meterProviderOnce struct {
	once     sync.Once
	provider metric.MeterProvider
	err      error
}

// MeterProvider exposes OpenTelemetry instruments on this registry's own
// endpoint, for libraries that report through OTel rather than the Prometheus
// client, the Temporal SDK among them. Their series land beside everything else
// on /metrics instead of needing a second exporter. The exporter registers a
// collector, so it is built once per registry no matter how often this is asked.
func (m *Registry) MeterProvider() (metric.MeterProvider, error) {
	if !m.enabled {
		return noop.NewMeterProvider(), nil
	}

	m.otel.once.Do(func() {
		exporter, err := otelprom.New(otelprom.WithRegisterer(m.registry))
		if err != nil {
			m.otel.err = fmt.Errorf("bridge otel metrics to prometheus: %w", err)
			return
		}
		m.otel.provider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	})

	return m.otel.provider, m.otel.err
}
