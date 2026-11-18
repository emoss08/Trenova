package telemetry

import "github.com/emoss08/trenova/operator/internal/instanceapi"

func (r *Relay) Kept() []instanceapi.TelemetrySample {
	return append([]instanceapi.TelemetrySample(nil), r.kept...)
}
