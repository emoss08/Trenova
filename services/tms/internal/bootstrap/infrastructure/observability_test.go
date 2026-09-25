package infrastructure

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestObservabilityModule_BuildsTheTracerWhetherOrNotAnythingAsksForIt(t *testing.T) {
	t.Cleanup(func() { aitrace.SetSamplingRate(1) })

	rate := 0.5
	cfg := &config.Config{}
	cfg.Monitoring.Tracing.AISamplingRate = &rate

	app := fxtest.New(t,
		fx.NopLogger,
		fx.Supply(cfg, zap.NewNop()),
		ObservabilityModule,
	)
	app.RequireStart()
	app.RequireStop()

	assert.InDelta(t, 0.5, aitrace.SamplingRate(), 0,
		"a worker that never injects the tracer still installs it")
}
