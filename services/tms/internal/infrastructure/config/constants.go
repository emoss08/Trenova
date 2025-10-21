package config

const (
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"
	EnvTest        = "test"
)

var ValidEnvs = []string{EnvDevelopment, EnvStaging, EnvProduction, EnvTest}
