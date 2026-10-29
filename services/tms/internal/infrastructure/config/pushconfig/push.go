package pushconfig

import (
	"slices"
)

type Config struct {
	VAPIDPublicKey       string   `mapstructure:"vapidPublicKey"`
	VAPIDPrivateKey      string   `mapstructure:"vapidPrivateKey"`
	Subject              string   `mapstructure:"subject"              validate:"omitempty"`
	AllowedEndpointHosts []string `mapstructure:"allowedEndpointHosts"`
}

var defaultPushEndpointHosts = []string{
	"fcm.googleapis.com",
	"updates.push.services.mozilla.com",
	"*.push.services.mozilla.com",
	"*.notify.windows.com",
	"web.push.apple.com",
	"*.push.apple.com",
}

func DefaultEndpointHosts() []string {
	return slices.Clone(defaultPushEndpointHosts)
}

func (c *Config) GetAllowedEndpointHosts() []string {
	if c == nil || len(c.AllowedEndpointHosts) == 0 {
		return DefaultEndpointHosts()
	}
	return slices.Clone(c.AllowedEndpointHosts)
}

func (c *Config) Enabled() bool {
	return c.VAPIDPublicKey != "" && c.VAPIDPrivateKey != ""
}
