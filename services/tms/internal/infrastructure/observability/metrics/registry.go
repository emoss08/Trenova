//revive:disable-next-line:var-naming
package metrics

import (
	"net/http"

	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/auditmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/documentmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Registry struct {
	registry *prometheus.Registry
	logger   *zap.Logger
	enabled  bool
	otel     meterProviderOnce
	genAI    genAIOnce

	HTTP       *httpmetrics.HTTP
	Error      *httpmetrics.Error
	Database   *platformmetrics.Database
	Temporal   *platformmetrics.Temporal
	Assistant  *aimetrics.Assistant
	Audit      *auditmetrics.Audit
	Document   *documentmetrics.Document
	EDI        *documentmetrics.EDI
	Report     *documentmetrics.Report
	GraphQL    *httpmetrics.GraphQL
	Permission *auditmetrics.Permission
	RateLimit  *httpmetrics.RateLimit
	Realtime   *platformmetrics.Realtime
	AIAudit    *aimetrics.AIAudit
	// ProposalPreview measures what people are shown before they approve.
	ProposalPreview *aimetrics.ProposalPreview
	Rules           *platformmetrics.WasmModule
	EDIModule       *platformmetrics.WasmModule
}

func graphQLOptions(cfg *monitoringconfig.Config) httpmetrics.GraphQLOptions {
	gqlCfg := &cfg.GraphQL
	return httpmetrics.GraphQLOptions{
		ResolverMetrics: gqlCfg.ResolverMetrics,
		MaxOperations:   gqlCfg.GetMaxTrackedOperations(),
	}
}

func NewRegistry(cfg *monitoringconfig.Config, logger *zap.Logger) (*Registry, error) {
	enabled := cfg.Metrics.Enabled

	if !enabled {
		logger.Warn("Metrics collection is disabled")
		dberror.SetConcurrencyObserver(nil)
		return &Registry{
			enabled:         false,
			logger:          logger,
			HTTP:            httpmetrics.NewHTTP(nil, logger, false),
			Error:           httpmetrics.NewError(nil, logger, false),
			Database:        platformmetrics.NewDatabase(nil, logger, false),
			Temporal:        platformmetrics.NewTemporal(nil, logger, false),
			Assistant:       aimetrics.NewAssistant(nil, logger, false),
			Audit:           auditmetrics.NewAudit(nil, logger, false),
			Document:        documentmetrics.NewDocument(nil, logger, false),
			EDI:             documentmetrics.NewEDI(nil, logger, false),
			Report:          documentmetrics.NewReport(nil, logger, false),
			GraphQL:         httpmetrics.NewGraphQL(nil, logger, false, graphQLOptions(cfg)),
			Permission:      auditmetrics.NewPermission(nil, logger, false),
			RateLimit:       httpmetrics.NewRateLimit(nil, logger, false),
			Realtime:        platformmetrics.NewRealtime(nil, logger, false),
			AIAudit:         aimetrics.NewAIAudit(nil, logger, false),
			ProposalPreview: aimetrics.NewProposalPreview(nil, logger, false),
			Rules: platformmetrics.NewWasmModule(
				nil,
				logger,
				false,
				platformmetrics.RulesSubsystem,
				"rules",
			),
			EDIModule: platformmetrics.NewWasmModule(
				nil,
				logger,
				false,
				platformmetrics.EDIModuleSubsystem,
				"EDI",
			),
		}, nil
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	m := &Registry{
		registry:        registry,
		logger:          logger,
		enabled:         true,
		HTTP:            httpmetrics.NewHTTP(registry, logger, true),
		Error:           httpmetrics.NewError(registry, logger, true),
		Database:        platformmetrics.NewDatabase(registry, logger, true),
		Temporal:        platformmetrics.NewTemporal(registry, logger, true),
		Assistant:       aimetrics.NewAssistant(registry, logger, true),
		Audit:           auditmetrics.NewAudit(registry, logger, true),
		Document:        documentmetrics.NewDocument(registry, logger, true),
		EDI:             documentmetrics.NewEDI(registry, logger, true),
		Report:          documentmetrics.NewReport(registry, logger, true),
		GraphQL:         httpmetrics.NewGraphQL(registry, logger, true, graphQLOptions(cfg)),
		Permission:      auditmetrics.NewPermission(registry, logger, true),
		RateLimit:       httpmetrics.NewRateLimit(registry, logger, true),
		Realtime:        platformmetrics.NewRealtime(registry, logger, true),
		AIAudit:         aimetrics.NewAIAudit(registry, logger, true),
		ProposalPreview: aimetrics.NewProposalPreview(registry, logger, true),
		Rules: platformmetrics.NewWasmModule(
			registry,
			logger,
			true,
			platformmetrics.RulesSubsystem,
			"rules",
		),
		EDIModule: platformmetrics.NewWasmModule(
			registry,
			logger,
			true,
			platformmetrics.EDIModuleSubsystem,
			"EDI",
		),
	}

	dberror.SetConcurrencyObserver(m.Database.RecordConcurrencyEvent)

	logger.Info("Metrics registry initialized",
		zap.Int("port", cfg.Metrics.Port),
		zap.String("path", cfg.Metrics.Path),
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
