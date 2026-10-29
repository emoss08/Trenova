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
	"go.uber.org/zap"
)

type Deps struct {
	Logger       *zap.Logger
	DB           kernelports.Transactor
	Connections  accountingconnectionports.AccountingConnectionRepository
	Apps         accountingconnectionports.AccountingAppCredentialRepository
	Integrations integrationports.IntegrationRepository
	Connectors   accountingconnectorports.AccountingConnectorRegistry
	Encryption   *encryptionservice.Service
	Audit        auditports.AuditService
	Realtime     realtimeports.RealtimeService
	Watchtower   watchtowerports.WatchtowerProjector
	Publisher    agenteventports.AgentEventPublisher
	Refresher    accountingconnectionports.AccountingReferenceRefresher
	Plans        platformplanports.PlanService
	Certificates accountingconnectionports.AccountingPendingCertificateRepository
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
func New(d Deps) *Kit {
	return &Kit{
		l:            d.Logger,
		db:           d.DB,
		connections:  d.Connections,
		apps:         d.Apps,
		integrations: d.Integrations,
		connectors:   d.Connectors,
		encryption:   d.Encryption,
		audit:        d.Audit,
		realtime:     d.Realtime,
		watchtower:   d.Watchtower,
		publisher:    d.Publisher,
		refresher:    d.Refresher,
		plans:        d.Plans,
		certificates: d.Certificates,
	}
}
