package accountingconnectionkit

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           kernelports.Transactor
	Connections  accountingconnectionports.AccountingConnectionRepository
	Apps         accountingconnectionports.AccountingAppCredentialRepository
	Integrations integrationports.IntegrationRepository
	Connectors   accountingconnectorports.AccountingConnectorRegistry
	Encryption   *encryptionservice.Service
	Audit        auditports.AuditService
	Realtime     realtimeports.RealtimeService                                    `optional:"true"`
	Watchtower   watchtowerports.WatchtowerProjector                              `optional:"true"`
	Publisher    agenteventports.AgentEventPublisher                              `optional:"true"`
	Refresher    accountingconnectionports.AccountingReferenceRefresher           `optional:"true"`
	Plans        platformplanports.PlanService                                    `optional:"true"`
	Certificates accountingconnectionports.AccountingPendingCertificateRepository `optional:"true"`
}

type Kit struct {
	l            *zap.Logger
	db           kernelports.Transactor
	connections  accountingconnectionports.AccountingConnectionRepository
	apps         accountingconnectionports.AccountingAppCredentialRepository
	integrations integrationports.IntegrationRepository
	connectors   accountingconnectorports.AccountingConnectorRegistry
	encryption   *encryptionservice.Service
	audit        auditports.AuditService
	realtime     realtimeports.RealtimeService
	watchtower   watchtowerports.WatchtowerProjector
	publisher    agenteventports.AgentEventPublisher
	refresher    accountingconnectionports.AccountingReferenceRefresher
	plans        platformplanports.PlanService
	certificates accountingconnectionports.AccountingPendingCertificateRepository
}

//nolint:gocritic // dependency injection
func New(p Params) *Kit {
	return &Kit{
		l:            p.Logger.Named("service.accounting-connection"),
		db:           p.DB,
		connections:  p.Connections,
		apps:         p.Apps,
		integrations: p.Integrations,
		connectors:   p.Connectors,
		encryption:   p.Encryption,
		audit:        p.Audit,
		realtime:     p.Realtime,
		watchtower:   p.Watchtower,
		publisher:    p.Publisher,
		refresher:    p.Refresher,
		plans:        p.Plans,
		certificates: p.Certificates,
	}
}
