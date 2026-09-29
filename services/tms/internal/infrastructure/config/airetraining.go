package config

import (
	"errors"
	"fmt"
	"time"
)

const (
	defaultAIRetrainingLookbackDays        = 365
	defaultAIRetrainingMinNewExamples      = 1000
	defaultAIRetrainingMinIntervalDays     = 28
	defaultAIRetrainingMaxPerOrganization  = 2000
	defaultAIRetrainingValidationPercent   = 10
	defaultAIRetrainingStructuredOutput    = "JSONSchema"
	defaultAIRetrainingMinAccuracyPercent  = 85
	defaultAIRetrainingMaxRegressionPoints = 0
	defaultAIRetrainingLease               = 2 * time.Hour
	minAIRetrainingLease                   = 10 * time.Minute
	maxAIRetrainingLease                   = 24 * time.Hour
)

var ErrAIRetrainingLease = fmt.Errorf(
	"aiRetraining.leaseDuration must be between %s and %s",
	minAIRetrainingLease,
	maxAIRetrainingLease,
)

var ErrAIRetrainingGate = errors.New(
	"aiRetraining.minAccuracyPercent and aiRetraining.maxRegressionPoints must be between 0 and 100",
)

type AIRetrainingConfig struct {
	Enabled              bool          `mapstructure:"enabled"`
	LookbackDays         int           `mapstructure:"lookbackDays"         validate:"omitempty,min=30,max=1095"`
	MinNewExamples       int           `mapstructure:"minNewExamples"       validate:"omitempty,min=1,max=1000000"`
	MinIntervalDays      int           `mapstructure:"minIntervalDays"      validate:"omitempty,min=1,max=365"`
	RetrainOnDrift       *bool         `mapstructure:"retrainOnDrift"`
	MaxPerOrganization   int           `mapstructure:"maxPerOrganization"   validate:"omitempty,min=1,max=20000"`
	ValidationPercent    int           `mapstructure:"validationPercent"    validate:"omitempty,min=1,max=50"`
	StructuredOutputMode string        `mapstructure:"structuredOutputMode" validate:"omitempty,oneof=JSONSchema JSONMode Prompted"`
	MinAccuracyPercent   *int          `mapstructure:"minAccuracyPercent"`
	MaxRegressionPoints  *int          `mapstructure:"maxRegressionPoints"`
	LeaseDuration        time.Duration `mapstructure:"leaseDuration"`
}

func (c *AIRetrainingConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}

func (c *AIRetrainingConfig) GetLookbackDays() int {
	if c == nil || c.LookbackDays <= 0 {
		return defaultAIRetrainingLookbackDays
	}

	return c.LookbackDays
}

func (c *AIRetrainingConfig) GetMinNewExamples() int {
	if c == nil || c.MinNewExamples <= 0 {
		return defaultAIRetrainingMinNewExamples
	}

	return c.MinNewExamples
}

func (c *AIRetrainingConfig) GetMinIntervalDays() int {
	if c == nil || c.MinIntervalDays <= 0 {
		return defaultAIRetrainingMinIntervalDays
	}

	return c.MinIntervalDays
}

func (c *AIRetrainingConfig) GetRetrainOnDrift() bool {
	return c == nil || c.RetrainOnDrift == nil || *c.RetrainOnDrift
}

func (c *AIRetrainingConfig) GetMaxPerOrganization() int {
	if c == nil || c.MaxPerOrganization <= 0 {
		return defaultAIRetrainingMaxPerOrganization
	}

	return c.MaxPerOrganization
}

func (c *AIRetrainingConfig) GetValidationPercent() int {
	if c == nil || c.ValidationPercent <= 0 {
		return defaultAIRetrainingValidationPercent
	}

	return c.ValidationPercent
}

func (c *AIRetrainingConfig) GetStructuredOutputMode() string {
	if c == nil || c.StructuredOutputMode == "" {
		return defaultAIRetrainingStructuredOutput
	}

	return c.StructuredOutputMode
}

func (c *AIRetrainingConfig) GetMinAccuracyPercent() int {
	if c == nil || c.MinAccuracyPercent == nil {
		return defaultAIRetrainingMinAccuracyPercent
	}

	return *c.MinAccuracyPercent
}

func (c *AIRetrainingConfig) GetMaxRegressionPoints() int {
	if c == nil || c.MaxRegressionPoints == nil {
		return defaultAIRetrainingMaxRegressionPoints
	}

	return *c.MaxRegressionPoints
}

func (c *AIRetrainingConfig) GetLeaseDuration() time.Duration {
	if c == nil || c.LeaseDuration <= 0 {
		return defaultAIRetrainingLease
	}

	return c.LeaseDuration
}

func validateAIRetrainingConfig(config *Config) error {
	retraining := &config.AIRetraining
	if lease := retraining.LeaseDuration; lease != 0 &&
		(lease < minAIRetrainingLease || lease > maxAIRetrainingLease) {
		return ErrAIRetrainingLease
	}
	if !percentInRange(retraining.MinAccuracyPercent) ||
		!percentInRange(retraining.MaxRegressionPoints) {
		return ErrAIRetrainingGate
	}

	return nil
}

func percentInRange(value *int) bool {
	return value == nil || (*value >= 0 && *value <= 100)
}
