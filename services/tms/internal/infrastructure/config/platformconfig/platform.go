package platformconfig

type Mode string

const (
	ModeCommunity   = Mode("community")
	ModeSelfHosted  = Mode("self_hosted")
	ModeDevelopment = Mode("development")
	ModeCloud       = Mode("cloud")
	ModeEnterprise  = Mode("enterprise")
)

type Config struct {
	Mode                  Mode     `mapstructure:"mode"                  validate:"omitempty,oneof=community self_hosted development cloud enterprise"`
	InstanceID            string   `mapstructure:"instanceId"`
	ReferenceDataStewards []string `mapstructure:"referenceDataStewards"`
}

func Defaults() map[string]any {
	return map[string]any{
		"mode": string(ModeSelfHosted),
	}
}

func (c *Config) IsCloud() bool {
	return c.GetMode() == ModeCloud
}

func (c *Config) GetMode() Mode {
	if c.Mode == "" {
		return ModeSelfHosted
	}

	if c.Mode == ModeCommunity || c.Mode == ModeEnterprise {
		return ModeSelfHosted
	}

	return c.Mode
}

func (c *Config) IsDevelopmentDeployment() bool {
	return c.GetMode() == ModeDevelopment
}
