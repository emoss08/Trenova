package metrics

import (
	"net/http"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Registry struct {
	registry *prometheus.Registry
	cfg      *config.Config
	logger   *zap.Logger
	enabled  bool

	HTTP     *HTTP
	Error    *Error
	Database *Database
	Temporal *Temporal
	Audit    *Audit
}

func NewRegistry(cfg *config.Config, logger *zap.Logger) (*Registry, error) {
	enabled := cfg.GetMetricsConfig().Enabled

	if !enabled {
		logger.Warn("Metrics collection is disabled")
		dberror.SetConcurrencyObserver(nil)
		return &Registry{
			enabled:  false,
			cfg:      cfg,
			logger:   logger,
			HTTP:     NewHTTP(nil, logger, false),
			Error:    NewError(nil, logger, false),
			Database: NewDatabase(nil, logger, false),
			Temporal: NewTemporal(nil, logger, false),
			Audit:    NewAudit(nil, logger, false),
		}, nil
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Registry{
		registry: registry,
		cfg:      cfg,
		logger:   logger,
		enabled:  true,
		HTTP:     NewHTTP(registry, logger, true),
		Error:    NewError(registry, logger, true),
		Database: NewDatabase(registry, logger, true),
		Temporal: NewTemporal(registry, logger, true),
		Audit:    NewAudit(registry, logger, true),
	}

	dberror.SetConcurrencyObserver(m.Database.RecordConcurrencyEvent)

	logger.Info("Metrics registry initialized",
		zap.Int("port", cfg.Monitoring.Metrics.Port),
		zap.String("path", cfg.Monitoring.Metrics.Path),
	)

	return m, nil
}

func (m *Registry) IsEnabled() bool {
	return m.enabled
}

func (m *Registry) Registry() *prometheus.Registry {
	return m.registry
}

func (m *Registry) Handler() http.Handler {
	if !m.IsEnabled() {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Metrics collection is disabled"))
		})
	}
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

func (m *Registry) GinMiddleware() gin.HandlerFunc {
	return m.HTTP.GinMiddleware()
}

func (m *Registry) RecordHTTPRequest(
	method, path string,
	status int,
	duration float64,
	responseSize int,
) {
	m.HTTP.RecordHTTPRequest(method, path, status, duration, responseSize)
}

func (m *Registry) IncrementActiveRequests() {
	m.HTTP.IncrementActiveRequests()
}

func (m *Registry) DecrementActiveRequests() {
	m.HTTP.DecrementActiveRequests()
}

func (m *Registry) RecordError(errorType, source string) {
	m.Error.RecordError(errorType, source)
}

func (m *Registry) RecordPanicRecovery() {
	m.Error.RecordPanicRecovery()
}
