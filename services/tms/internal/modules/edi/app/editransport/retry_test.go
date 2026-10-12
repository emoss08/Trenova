package editransport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeliveryRetryPolicyFromConfig(t *testing.T) {
	t.Parallel()

	assert.Nil(t, DeliveryRetryPolicyFromConfig(nil))
	assert.Nil(t, DeliveryRetryPolicyFromConfig(map[string]any{"host": "sftp.example"}))

	policy := DeliveryRetryPolicyFromConfig(map[string]any{
		ConfigKeyRetryMaxAttempts:            "3",
		ConfigKeyRetryInitialIntervalSeconds: 60,
		ConfigKeyRetryMaxIntervalSeconds:     float64(600),
	})
	require.NotNil(t, policy)
	assert.Equal(t, int32(3), policy.MaxAttempts)
	assert.Equal(t, int64(60), policy.InitialIntervalSeconds)
	assert.Equal(t, int64(600), policy.MaxIntervalSeconds)

	partial := DeliveryRetryPolicyFromConfig(map[string]any{
		ConfigKeyRetryMaxAttempts: 10,
	})
	require.NotNil(t, partial)
	assert.Equal(t, int32(10), partial.MaxAttemptsOrDefault())
	assert.Equal(t, DefaultDeliveryInitialInterval, partial.InitialIntervalOrDefault())
	assert.Equal(t, DefaultDeliveryMaxInterval, partial.MaxIntervalOrDefault())
}

func TestDeliveryRetryPolicyDefaults(t *testing.T) {
	t.Parallel()

	var policy *DeliveryRetryPolicy
	assert.Equal(t, DefaultDeliveryMaxAttempts, policy.MaxAttemptsOrDefault())
	assert.Equal(t, DefaultDeliveryInitialInterval, policy.InitialIntervalOrDefault())
	assert.Equal(t, DefaultDeliveryMaxInterval, policy.MaxIntervalOrDefault())

	inverted := &DeliveryRetryPolicy{
		InitialIntervalSeconds: 300,
		MaxIntervalSeconds:     60,
	}
	assert.Equal(t, 300*time.Second, inverted.MaxIntervalOrDefault())
}
