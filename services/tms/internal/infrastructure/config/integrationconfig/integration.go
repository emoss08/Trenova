package integrationconfig

import (
	"github.com/emoss08/trenova/shared/httpsafe"
)

type Config struct {
	AllowPrivateNetworks bool `mapstructure:"allowPrivateNetworks"`
}

func (c *Config) EgressPolicy() httpsafe.Policy {
	return httpsafe.Policy{AllowPrivateNetworks: c != nil && c.AllowPrivateNetworks}
}
