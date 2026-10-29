package appconfig

import "strings"

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

var ValidEnvs = []string{EnvDevelopment, EnvStaging, EnvProduction, EnvTest}

type Config struct {
	Name               string `mapstructure:"name"               validate:"required,min=1,max=100"`
	Env                string `mapstructure:"env"                validate:"required,oneof=development staging production test"`
	Debug              bool   `mapstructure:"debug"`
	Version            string `mapstructure:"version"            validate:"required"`
	ProblemTypeBaseURI string `mapstructure:"problemTypeBaseUri"`
	WebBaseURL         string `mapstructure:"webBaseUrl"         validate:"omitempty,url"`
}

func Defaults(env string) map[string]any {
	return map[string]any{
		"env":     env,
		"debug":   false,
		"version": "0.0.0",
	}
}

func (c *Config) GetWebBaseURL() string {
	return strings.TrimSuffix(c.WebBaseURL, "/")
}

func (c *Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

func (c *Config) IsProduction() bool { return c.Env == EnvProduction }

func (c *Config) IsStaging() bool { return c.Env == EnvStaging }

func (c *Config) IsTest() bool { return c.Env == EnvTest }

func (c *Config) GetProblemTypeBaseURI() string {
	if c.ProblemTypeBaseURI != "" {
		return strings.TrimSuffix(c.ProblemTypeBaseURI, "/") + "/"
	}

	return "https://api.trenova.app/problems/"
}
