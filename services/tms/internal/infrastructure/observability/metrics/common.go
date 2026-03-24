package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	Namespace    = "trenova"
	UnknownValue = "unknown"
)

var (
	HTTPDurationBuckets     = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	HTTPResponseSizeBuckets = prometheus.ExponentialBuckets(100, 10, 7)
)

type Base struct {
	registry *prometheus.Registry
	logger   *zap.Logger
	enabled  bool
}

func NewBase(registry *prometheus.Registry, logger *zap.Logger, enabled bool) Base {
	return Base{
		registry: registry,
		logger:   logger,
		enabled:  enabled,
	}
}

func (b *Base) IsEnabled() bool {
	return b.enabled
}

func (b *Base) ifEnabled(fn func()) {
	if b.enabled {
		fn()
	}
}

func (b *Base) mustRegister(collectors ...prometheus.Collector) {
	for _, c := range collectors {
		b.registry.MustRegister(c)
	}
}
