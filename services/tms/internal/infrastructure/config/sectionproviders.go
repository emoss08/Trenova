package config

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/accountingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/auditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/cacheconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/carrierintelligenceconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/contentscanconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/ediconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/imagingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/importsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/portalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/pushconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/rendererconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/systemconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/twilioconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/updateconfig"
	"go.uber.org/fx"
)

var SectionProviders = fx.Provide(
	func(c *Config) *appconfig.Config { return &c.App },
	func(c *Config) *databaseconfig.Config { return &c.Database },
	func(c *Config) *monitoringconfig.Config { return &c.Monitoring },
	func(c *Config) *cacheconfig.Config { return &c.Cache },
	func(c *Config) *serverconfig.Config { return &c.Server },
	func(c *Config) *securityconfig.Config { return &c.Security },
	func(c *Config) *loggingconfig.Config { return &c.Logging },
	func(c *Config) *temporalconfig.Config { return &c.Temporal },
	func(c *Config) *storageconfig.Config { return &c.Storage },
	func(c *Config) *systemconfig.Config { return &c.System },
	func(c *Config) *realtimeconfig.Config { return &c.Realtime },
	func(c *Config) *searchconfig.Config { return &c.Search },
	func(c *Config) *aiconfig.Config { return &c.AI },
	func(c *Config) *auditconfig.Config { return &c.Audit },
	func(c *Config) *aiauditconfig.Config { return &c.AIAudit },
	func(c *Config) *updateconfig.Config { return &c.Update },
	func(c *Config) *twilioconfig.Config { return &c.Twilio },
	func(c *Config) *platformconfig.Config { return &c.Platform },
	func(c *Config) *reportingconfig.Config { return &c.Reporting },
	func(c *Config) *rendererconfig.Config { return &c.Renderer },
	func(c *Config) *documentconversionconfig.Config { return &c.DocumentConversion },
	func(c *Config) *imagingconfig.Config { return &c.Imaging },
	func(c *Config) *portalconfig.Config { return &c.Portal },
	func(c *Config) *pushconfig.Config { return &c.Push },
	func(c *Config) *tenderingconfig.Config { return &c.Tendering },
	func(c *Config) *carrierintelligenceconfig.Config { return &c.CarrierIntelligence },
	func(c *Config) *accountingconfig.Config { return &c.Accounting },
	func(c *Config) *ediconfig.Config { return &c.EDI },
	func(c *Config) *importsconfig.Config { return &c.Imports },
	func(c *Config) *contentscanconfig.Config { return &c.ContentScan },
)
