package config

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIRetrainingConfig_Defaults(t *testing.T) {
	t.Parallel()

	for _, cfg := range []*AIRetrainingConfig{nil, {}} {
		assert.False(t, cfg.IsEnabled())
		assert.Equal(t, 365, cfg.GetLookbackDays())
		assert.Equal(t, 1000, cfg.GetMinNewExamples())
		assert.Equal(t, 28, cfg.GetMinIntervalDays())
		assert.True(t, cfg.GetRetrainOnDrift())
		assert.Equal(t, 2000, cfg.GetMaxPerOrganization())
		assert.Equal(t, 10, cfg.GetValidationPercent())
		assert.Equal(t, "JSONSchema", cfg.GetStructuredOutputMode())
		assert.Equal(t, 85, cfg.GetMinAccuracyPercent())
		assert.Equal(t, 0, cfg.GetMaxRegressionPoints())
		assert.Equal(t, 2*time.Hour, cfg.GetLeaseDuration())
	}
}

func TestAIRetrainingConfig_HonoursExplicitZeroes(t *testing.T) {
	t.Parallel()

	off, zero, three := false, 0, 3
	cfg := &AIRetrainingConfig{
		Enabled:             true,
		RetrainOnDrift:      &off,
		MinAccuracyPercent:  &zero,
		MaxRegressionPoints: &three,
	}

	assert.True(t, cfg.IsEnabled())
	assert.False(t, cfg.GetRetrainOnDrift())
	assert.Equal(t, 0, cfg.GetMinAccuracyPercent(), "an explicit zero turns the accuracy floor off")
	assert.Equal(t, 3, cfg.GetMaxRegressionPoints())
}

func TestValidateAIRetrainingConfig(t *testing.T) {
	t.Parallel()

	negative, over := -1, 101
	tests := []struct {
		name   string
		mutate func(*AIRetrainingConfig)
		want   error
	}{
		{name: "empty section", mutate: func(*AIRetrainingConfig) {}},
		{
			name:   "lease too short",
			mutate: func(c *AIRetrainingConfig) { c.LeaseDuration = 5 * time.Minute },
			want:   ErrAIRetrainingLease,
		},
		{
			name:   "lease too long",
			mutate: func(c *AIRetrainingConfig) { c.LeaseDuration = 25 * time.Hour },
			want:   ErrAIRetrainingLease,
		},
		{
			name:   "lease in range",
			mutate: func(c *AIRetrainingConfig) { c.LeaseDuration = 30 * time.Minute },
		},
		{
			name:   "negative accuracy floor",
			mutate: func(c *AIRetrainingConfig) { c.MinAccuracyPercent = &negative },
			want:   ErrAIRetrainingGate,
		},
		{
			name:   "regression above 100 points",
			mutate: func(c *AIRetrainingConfig) { c.MaxRegressionPoints = &over },
			want:   ErrAIRetrainingGate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{}
			tt.mutate(&cfg.AIRetraining)
			err := validateAIRetrainingConfig(cfg)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestValidateConfig_BoundsTheRetrainingPolicy(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.AIRetraining.StructuredOutputMode = "Loose"
	err := NewLoader().validateConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "structuredoutputmode")

	cfg = newValidConfig()
	cfg.AIRetraining.LookbackDays = 7
	err = NewLoader().validateConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookbackdays must be at least 30")
}

var (
	alertSecretKey   = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	validAlertSecret = aiRetrainingAlertSecretPrefix + alertSecretKey
)

func TestValidateAIRetrainingAlerts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		alerts AIRetrainingAlertsConfig
		want   error
	}{
		{name: "no webhook"},
		{name: "https webhook", alerts: AIRetrainingAlertsConfig{WebhookURL: "https://hooks.slack.com/services/T/B/x"}},
		{
			name:   "signed webhook",
			alerts: AIRetrainingAlertsConfig{WebhookURL: "https://alerts.example.com/in", Secret: validAlertSecret},
		},
		{
			name:   "not a url",
			alerts: AIRetrainingAlertsConfig{WebhookURL: "alerts.example.com/in"},
			want:   ErrAIRetrainingAlertURL,
		},
		{
			name:   "another scheme",
			alerts: AIRetrainingAlertsConfig{WebhookURL: "ftp://alerts.example.com/in"},
			want:   ErrAIRetrainingAlertURL,
		},
		{
			name:   "secret without prefix",
			alerts: AIRetrainingAlertsConfig{Secret: alertSecretKey},
			want:   ErrAIRetrainingAlertSecret,
		},
		{
			name:   "secret too short",
			alerts: AIRetrainingAlertsConfig{Secret: "whsec_c2hvcnQ="},
			want:   ErrAIRetrainingAlertSecret,
		},
		{
			name:   "secret not base64",
			alerts: AIRetrainingAlertsConfig{Secret: "whsec_not base64 at all, not at all, not at all"},
			want:   ErrAIRetrainingAlertSecret,
		},
		{
			name:   "timeout too long",
			alerts: AIRetrainingAlertsConfig{Timeout: 2 * time.Minute},
			want:   ErrAIRetrainingAlertTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			alerts := tt.alerts
			err := validateAIRetrainingAlerts(&alerts)
			if tt.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestAIRetrainingAlertsInProduction(t *testing.T) {
	t.Parallel()

	cfg := &Config{}
	require.NoError(t, validateAIRetrainingAlertsProduction(cfg), "no webhook is fine")

	cfg.AIRetraining.Alerts = AIRetrainingAlertsConfig{WebhookURL: "https://hooks.slack.com/services/T/B/x"}
	require.NoError(t, validateAIRetrainingAlertsProduction(cfg),
		"a receiver that cannot verify a signature, such as Slack, needs no secret")

	cfg.AIRetraining.Alerts = AIRetrainingAlertsConfig{
		WebhookURL: "http://alerts.example.com/in",
		Secret:     validAlertSecret,
	}
	require.ErrorIs(t, validateAIRetrainingAlertsProduction(cfg), ErrAIRetrainingAlertInsecure,
		"a retraining alert names model paths and should not cross the network in clear")

	cfg.AIRetraining.Alerts.WebhookURL = "https://alerts.example.com/in"
	require.NoError(t, validateAIRetrainingAlertsProduction(cfg))
}

func TestAIRetrainingAlertsNeverPrintTheirSecret(t *testing.T) {
	t.Parallel()

	alerts := AIRetrainingAlertsConfig{WebhookURL: "https://alerts.example.com/in", Secret: validAlertSecret}
	assert.NotContains(t, alerts.String(), alertSecretKey)
	assert.NotContains(t, fmt.Sprintf("%v %+v %#v", alerts, alerts, alerts), alertSecretKey)
	assert.Contains(t, alerts.String(), redactedValue)
	assert.Equal(t, 10*time.Second, alerts.GetTimeout())
	assert.True(t, alerts.Enabled())
	assert.True(t, alerts.Signed())
}
