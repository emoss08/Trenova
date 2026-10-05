package cloudconfig

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/urlutils"
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
	defaultAIRetrainingAlertTimeout        = 10 * time.Second
	minAIRetrainingAlertTimeout            = time.Second
	maxAIRetrainingAlertTimeout            = time.Minute
	minAIRetrainingAlertSecretBytes        = 24
	aiRetrainingAlertSecretPrefix          = "whsec_"
	redactedValue                          = "[redacted]"
)

var ErrAIRetrainingLease = fmt.Errorf(
	"aiRetraining.leaseDuration must be between %s and %s",
	minAIRetrainingLease,
	maxAIRetrainingLease,
)

var ErrAIRetrainingGate = errors.New(
	"aiRetraining.minAccuracyPercent and aiRetraining.maxRegressionPoints must be between 0 and 100",
)

var (
	ErrAIRetrainingAlertURL = errors.New(
		"aiRetraining.alerts.webhookUrl must be an absolute http or https URL",
	)
	ErrAIRetrainingAlertSecret = fmt.Errorf(
		"aiRetraining.alerts.secret must be %s followed by at least %d base64-encoded bytes",
		aiRetrainingAlertSecretPrefix,
		minAIRetrainingAlertSecretBytes,
	)
	ErrAIRetrainingAlertTimeout = fmt.Errorf(
		"aiRetraining.alerts.timeout must be between %s and %s",
		minAIRetrainingAlertTimeout,
		maxAIRetrainingAlertTimeout,
	)
	ErrAIRetrainingAlertInsecure = errors.New(
		"production and staging require aiRetraining.alerts.webhookUrl to use https",
	)
)

type AIRetrainingConfig struct {
	Enabled              bool                     `mapstructure:"enabled"`
	LookbackDays         int                      `mapstructure:"lookbackDays"         validate:"omitempty,min=30,max=1095"`
	MinNewExamples       int                      `mapstructure:"minNewExamples"       validate:"omitempty,min=1,max=1000000"`
	MinIntervalDays      int                      `mapstructure:"minIntervalDays"      validate:"omitempty,min=1,max=365"`
	RetrainOnDrift       *bool                    `mapstructure:"retrainOnDrift"`
	MaxPerOrganization   int                      `mapstructure:"maxPerOrganization"   validate:"omitempty,min=1,max=20000"`
	ValidationPercent    int                      `mapstructure:"validationPercent"    validate:"omitempty,min=1,max=50"`
	StructuredOutputMode string                   `mapstructure:"structuredOutputMode" validate:"omitempty,oneof=JSONSchema JSONMode Prompted"`
	MinAccuracyPercent   *int                     `mapstructure:"minAccuracyPercent"`
	MaxRegressionPoints  *int                     `mapstructure:"maxRegressionPoints"`
	LeaseDuration        time.Duration            `mapstructure:"leaseDuration"`
	Alerts               AIRetrainingAlertsConfig `mapstructure:"alerts"`
}

type AIRetrainingAlertsConfig struct {
	WebhookURL          string        `mapstructure:"webhookUrl"`
	Secret              string        `mapstructure:"secret"`
	IncludeSkipped      bool          `mapstructure:"includeSkipped"`
	AllowPrivateNetwork bool          `mapstructure:"allowPrivateNetwork"`
	Timeout             time.Duration `mapstructure:"timeout"`
}

func (c AIRetrainingAlertsConfig) String() string {
	secret := ""
	if c.Secret != "" {
		secret = redactedValue
	}

	return fmt.Sprintf(
		"{WebhookURL:%s Secret:%s IncludeSkipped:%t AllowPrivateNetwork:%t Timeout:%s}",
		c.WebhookURL,
		secret,
		c.IncludeSkipped,
		c.AllowPrivateNetwork,
		c.Timeout,
	)
}

func (c AIRetrainingAlertsConfig) GoString() string {
	return "cloudconfig.AIRetrainingAlertsConfig" + c.String()
}

func (c *AIRetrainingAlertsConfig) Enabled() bool {
	return c != nil && strings.TrimSpace(c.WebhookURL) != ""
}

func (c *AIRetrainingAlertsConfig) Signed() bool {
	return c != nil && c.Secret != ""
}

func (c *AIRetrainingAlertsConfig) GetTimeout() time.Duration {
	if c == nil || c.Timeout <= 0 {
		return defaultAIRetrainingAlertTimeout
	}

	return c.Timeout
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

func validateAIRetrainingConfig(_ *config.Config, settings *Settings) error {
	retraining := &settings.AIRetraining
	if lease := retraining.LeaseDuration; lease != 0 &&
		(lease < minAIRetrainingLease || lease > maxAIRetrainingLease) {
		return ErrAIRetrainingLease
	}
	if !percentInRange(retraining.MinAccuracyPercent) ||
		!percentInRange(retraining.MaxRegressionPoints) {
		return ErrAIRetrainingGate
	}

	return validateAIRetrainingAlerts(&retraining.Alerts)
}

func validateAIRetrainingAlerts(alerts *AIRetrainingAlertsConfig) error {
	if timeout := alerts.Timeout; timeout != 0 &&
		(timeout < minAIRetrainingAlertTimeout || timeout > maxAIRetrainingAlertTimeout) {
		return ErrAIRetrainingAlertTimeout
	}
	if alerts.Signed() {
		key, err := base64.StdEncoding.DecodeString(
			strings.TrimPrefix(alerts.Secret, aiRetrainingAlertSecretPrefix),
		)
		if !strings.HasPrefix(alerts.Secret, aiRetrainingAlertSecretPrefix) || err != nil ||
			len(key) < minAIRetrainingAlertSecretBytes {
			return ErrAIRetrainingAlertSecret
		}
	}
	if !alerts.Enabled() {
		return nil
	}

	if _, ok := urlutils.ParseAbsoluteHTTP(alerts.WebhookURL); !ok {
		return ErrAIRetrainingAlertURL
	}

	return nil
}

func validateAIRetrainingAlertsProduction(settings *Settings) error {
	alerts := &settings.AIRetraining.Alerts
	if !alerts.Enabled() {
		return nil
	}
	if !strings.HasPrefix(strings.TrimSpace(alerts.WebhookURL), "https://") {
		return ErrAIRetrainingAlertInsecure
	}

	return nil
}

func percentInRange(value *int) bool {
	return value == nil || (*value >= 0 && *value <= 100)
}
