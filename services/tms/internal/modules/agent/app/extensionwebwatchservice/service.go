package extensionwebwatchservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentextensionservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/secretconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type webWatchRuntime interface {
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
	Settle(
		ctx context.Context,
		call agentextensionservice.MeteredCall,
		outcome agentextensionservice.SettledCall,
	)
	ProviderError(f *agentextensionservice.ProviderFailure) error
}

type Params struct {
	fx.In

	Logger     *zap.Logger
	Extensions *agentextensionservice.Service
	Encryption *encryptionservice.Service
	Connectors []agentextensionports.ExtensionConnector `group:"agent_extension_connectors"`
	Watches    agentextensionports.WebWatchRepository
	Findings   agentextensionports.WebWatchFindingRepository
	Workflows  kernelports.WorkflowStarter `                                   optional:"true"`
	AppConfig  *appconfig.Config           `                                   optional:"true"`
}

type Service struct {
	l         *zap.Logger
	core      webWatchRuntime
	secrets   secretconfig.Codec
	watchers  map[agentextension.Type]agentextensionports.WebWatchProvider
	watches   agentextensionports.WebWatchRepository
	findings  agentextensionports.WebWatchFindingRepository
	workflows kernelports.WorkflowStarter
	webBase   string
	now       func() int64
}

func New(p Params) *Service {
	watchers := make(
		map[agentextension.Type]agentextensionports.WebWatchProvider,
		len(p.Connectors),
	)
	for _, connector := range p.Connectors {
		if provider, ok := connector.(agentextensionports.WebWatchProvider); ok {
			watchers[connector.Type()] = provider
		}
	}

	return &Service{
		l:         p.Logger.Named("service.agentextension"),
		core:      p.Extensions,
		secrets:   secretconfig.NewCodec(p.Encryption, p.Logger),
		watchers:  watchers,
		watches:   p.Watches,
		findings:  p.Findings,
		workflows: p.Workflows,
		webBase:   webBaseOf(p.AppConfig),
		now:       timeutils.NowUnix,
	}
}
