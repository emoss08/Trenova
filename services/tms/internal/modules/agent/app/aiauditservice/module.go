package aiauditservice

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Ledger        aiauditports.AIAuditRepository
	Source        aiauditports.AIAuditSourceRepository
	ExportRepo    aiauditports.AIAuditExportRepository
	Retention     tenantports.DataRetentionRepository
	Policies      *agenttoolpolicy.Catalog
	Registry      *permission.Registry
	Permissions   permissionports.PermissionEngine
	Storage       kernelports.Client
	Workflows     kernelports.WorkflowStarter
	Notifications *notificationservice.Service
	Realtime      agentports.RealtimeService `optional:"true"`
	Audit         auditports.AuditService
	Metrics       *metrics.Registry `optional:"true"`
	Config        *config.Config
	Logger        *zap.Logger
}

// Components are the trail's parts: the service the API reads through and
// the pieces the jobs drive.
type Components struct {
	fx.Out

	Service   aiauditports.AIAuditService
	Projector *Projector
	Verifier  *Verifier
	Exports   *Exports
	Retention *Retention
	Keyring   *Keyring
}

func aiAuditMetrics(registry *metrics.Registry) *metrics.AIAudit {
	if registry == nil {
		return nil
	}

	return registry.AIAudit
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) Components {
	keyring := NewKeyring(&p.Config.AIAudit.Chain, p.Logger)
	registryMetrics := aiAuditMetrics(p.Metrics)
	masker := auditservice.NewSensitiveDataManager(p.Config.Security.Encryption)
	redactor := NewRedactor(p.Registry, p.Policies, masker)

	projector := NewProjector(&ProjectorParams{
		Ledger:    p.Ledger,
		Source:    p.Source,
		Keyring:   keyring,
		Redactor:  redactor,
		Metrics:   registryMetrics,
		BatchSize: p.Config.AIAudit.Projector.GetBatchSize(),
		Logger:    p.Logger,
	})
	verifier := NewVerifier(&VerifierParams{
		Ledger:   p.Ledger,
		Keyring:  keyring,
		Notifier: p.Notifications,
		Audit:    p.Audit,
		Realtime: p.Realtime,
		Metrics:  registryMetrics,
		Logger:   p.Logger,
	})
	exports := NewExports(&ExportsParams{
		Ledger:      p.Ledger,
		Exports:     p.ExportRepo,
		Source:      p.Source,
		Exporter:    NewExporter(p.Ledger, keyring),
		Storage:     p.Storage,
		Workflows:   p.Workflows,
		Notifier:    p.Notifications,
		Realtime:    p.Realtime,
		Audit:       p.Audit,
		Permissions: p.Permissions,
		Config:      &p.Config.AIAudit.Export,
		Metrics:     registryMetrics,
		Logger:      p.Logger,
	})

	return Components{
		Service: NewService(ServiceParams{
			Ledger:    p.Ledger,
			Projector: projector,
			Verifier:  verifier,
			Exports:   exports,
			Workflows: p.Workflows,
			TraceURL:  p.Config.Monitoring.Tracing.TraceURL,
			Logger:    p.Logger,
		}),
		Projector: projector,
		Verifier:  verifier,
		Exports:   exports,
		Retention: NewRetention(p.Ledger, p.Retention, registryMetrics, p.Logger),
		Keyring:   keyring,
	}
}

var Module = fx.Module("ai-audit-service", fx.Provide(New))
