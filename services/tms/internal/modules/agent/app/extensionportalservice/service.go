package extensionportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type portalRuntime interface {
	Serving(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		capability agentextension.Capability,
	) (*agentextensionservice.ExtensionSettings, error)
	SettingsFor(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		typ agentextension.Type,
	) (*agentextensionservice.ExtensionSettings, error)
	Reserve(ctx context.Context, call agentextensionservice.MeteredCall) (int, error)
	Settle(
		ctx context.Context,
		call agentextensionservice.MeteredCall,
		outcome agentextensionservice.SettledCall,
	)
	ProviderError(f *agentextensionservice.ProviderFailure) error
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Extensions   *agentextensionservice.Service
	Encryption   *encryptionservice.Service
	AuditService auditports.AuditService
	Connectors   []agentextensionports.ExtensionConnector `group:"agent_extension_connectors"`
	Sites        agentextensionports.PortalSiteRepository
	Runs         agentextensionports.PortalRunRepository
	Workflows    kernelports.WorkflowStarter `                                   optional:"true"`
	AppConfig    *appconfig.Config           `                                   optional:"true"`
}

type Service struct {
	l         *zap.Logger
	core      portalRuntime
	secrets   secretconfig.Codec
	audit     auditports.AuditService
	browsers  map[agentextension.Type]agentextensionports.BrowserProvider
	sites     agentextensionports.PortalSiteRepository
	runs      agentextensionports.PortalRunRepository
	workflows kernelports.WorkflowStarter
	appHost   string
	now       func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	browsers := make(
		map[agentextension.Type]agentextensionports.BrowserProvider,
		len(p.Connectors),
	)
	for _, connector := range p.Connectors {
		if provider, ok := connector.(agentextensionports.BrowserProvider); ok {
			browsers[connector.Type()] = provider
		}
	}

	return &Service{
		l:         p.Logger.Named("service.agentextension"),
		core:      p.Extensions,
		secrets:   secretconfig.NewCodec(p.Encryption, p.Logger),
		audit:     p.AuditService,
		browsers:  browsers,
		sites:     p.Sites,
		runs:      p.Runs,
		workflows: p.Workflows,
		appHost:   appHostOf(p.AppConfig),
		now:       timeutils.NowUnix,
	}
}
