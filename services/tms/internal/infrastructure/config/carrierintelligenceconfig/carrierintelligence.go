package carrierintelligenceconfig

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

type Config struct {
	AllowedHosts    []string      `mapstructure:"allowedHosts"`
	SandboxAllowed  *bool         `mapstructure:"sandboxAllowed"`
	InteractiveWait time.Duration `mapstructure:"interactiveWait"`
	ReplicaHint     int           `mapstructure:"replicaHint"     validate:"omitempty,min=1,max=1000"`
}

var defaultCarrierIntelligenceHosts = []string{"api.carrierok.com", "mobile.fmcsa.dot.gov"}

func (c *Config) GetAllowedHosts() []string {
	if len(c.AllowedHosts) == 0 {
		return defaultCarrierIntelligenceHosts
	}
	return c.AllowedHosts
}

func (c *Config) IsSandboxAllowed(app *appconfig.Config) bool {
	if c.SandboxAllowed != nil {
		return *c.SandboxAllowed
	}
	return !app.IsProduction()
}

func (c *Config) GetInteractiveWait() time.Duration {
	if c.InteractiveWait <= 0 {
		return 8 * time.Second
	}
	return c.InteractiveWait
}

func (c *Config) GetReplicaHint() int {
	if c.ReplicaHint <= 0 {
		return 1
	}
	return c.ReplicaHint
}
