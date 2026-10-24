package common

import (
	"time"
)

type Environment string

const (
	EnvDevelopment = Environment("development")
	EnvTest        = Environment("test")
	EnvStaging     = Environment("staging")
	EnvProduction  = Environment("production")
)

type OperationOptions struct {
	DryRun      bool
	Force       bool
	Interactive bool
	Verbose     bool
	Target      string
	Environment Environment
}

type SeedStatus struct {
	Name        string
	Version     string
	AppliedAt   time.Time
	Checksum    string
	Environment Environment
	Status      string
}
