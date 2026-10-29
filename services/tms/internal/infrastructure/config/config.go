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
)

type Config struct {
	App                 appconfig.Config                 `mapstructure:"app"                 validate:"required"`
	Database            databaseconfig.Config            `mapstructure:"database"            validate:"required"`
	Monitoring          monitoringconfig.Config          `mapstructure:"monitoring"          validate:"required"`
	Cache               cacheconfig.Config               `mapstructure:"cache"               validate:"required"`
	Server              serverconfig.Config              `mapstructure:"server"              validate:"required"`
	Security            securityconfig.Config            `mapstructure:"security"            validate:"required"`
	Logging             loggingconfig.Config             `mapstructure:"logging"             validate:"required"`
	Temporal            temporalconfig.Config            `mapstructure:"temporal"            validate:"required"`
	Storage             storageconfig.Config             `mapstructure:"storage"             validate:"required"`
	System              systemconfig.Config              `mapstructure:"system"              validate:"required"`
	Realtime            realtimeconfig.Config            `mapstructure:"realtime"`
	Search              searchconfig.Config              `mapstructure:"search"`
	AI                  aiconfig.Config                  `mapstructure:"ai"`
	Audit               auditconfig.Config               `mapstructure:"audit"`
	AIAudit             aiauditconfig.Config             `mapstructure:"aiAudit"`
	Update              updateconfig.Config              `mapstructure:"update"`
	Twilio              twilioconfig.Config              `mapstructure:"twilio"`
	Platform            platformconfig.Config            `mapstructure:"platform"`
	Reporting           reportingconfig.Config           `mapstructure:"reporting"`
	Renderer            rendererconfig.Config            `mapstructure:"renderer"`
	DocumentConversion  documentconversionconfig.Config  `mapstructure:"documentConversion"`
	Imaging             imagingconfig.Config             `mapstructure:"imaging"`
	Portal              portalconfig.Config              `mapstructure:"portal"`
	Push                pushconfig.Config                `mapstructure:"push"`
	Tendering           tenderingconfig.Config           `mapstructure:"tendering"`
	CarrierIntelligence carrierintelligenceconfig.Config `mapstructure:"carrierIntelligence"`
	Accounting          accountingconfig.Config          `mapstructure:"accounting"`
	EDI                 ediconfig.Config                 `mapstructure:"edi"`
	Imports             importsconfig.Config             `mapstructure:"imports"`
	ContentScan         contentscanconfig.Config         `mapstructure:"contentScan"`

	extensions map[string]any
}
