package telemetry

import "github.com/emoss08/trenova/operator/internal/instanceapi"

func (r *Relay) Kept() []instanceapi.TelemetrySample {
	return r.queue.Kept()
}
