package ediconfig

import (
	"github.com/emoss08/trenova/shared/httpsafe"
)

const DefaultMaxInboundFileBytes int64 = 32 << 20

type Config struct {
	AllowPrivateNetworks bool  `mapstructure:"allowPrivateNetworks"`
	MaxInboundFileBytes  int64 `mapstructure:"maxInboundFileBytes"  validate:"omitempty,min=1024"`
}

func (c *Config) GetMaxInboundFileBytes() int64 {
	if c == nil || c.MaxInboundFileBytes <= 0 {
		return DefaultMaxInboundFileBytes
	}
	return c.MaxInboundFileBytes
}

func (c *Config) EgressPolicy() httpsafe.Policy {
	return httpsafe.Policy{AllowPrivateNetworks: c != nil && c.AllowPrivateNetworks}
}
