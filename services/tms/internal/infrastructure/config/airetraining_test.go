package config

import (
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
