package editransport

import (
	"time"

	"github.com/emoss08/trenova/shared/maputils"
)

const (
	ConfigKeyRetryMaxAttempts            = "retryMaxAttempts"
	ConfigKeyRetryInitialIntervalSeconds = "retryInitialIntervalSeconds"
	ConfigKeyRetryMaxIntervalSeconds     = "retryMaxIntervalSeconds"

	DefaultDeliveryMaxAttempts     = int32(6)
	DefaultDeliveryInitialInterval = 30 * time.Second
	DefaultDeliveryMaxInterval     = 15 * time.Minute

	MinDeliveryMaxAttempts     = 1
	MaxDeliveryMaxAttempts     = 25
	MinDeliveryIntervalSeconds = 5
	MaxDeliveryIntervalSeconds = 24 * 60 * 60
)

type DeliveryRetryPolicy struct {
	MaxAttempts            int32 `json:"maxAttempts"`
	InitialIntervalSeconds int64 `json:"initialIntervalSeconds"`
	MaxIntervalSeconds     int64 `json:"maxIntervalSeconds"`
}

func (p *DeliveryRetryPolicy) MaxAttemptsOrDefault() int32 {
	if p == nil || p.MaxAttempts <= 0 {
		return DefaultDeliveryMaxAttempts
	}
	return p.MaxAttempts
}

func (p *DeliveryRetryPolicy) InitialIntervalOrDefault() time.Duration {
	if p == nil || p.InitialIntervalSeconds <= 0 {
		return DefaultDeliveryInitialInterval
	}
	return time.Duration(p.InitialIntervalSeconds) * time.Second
}

func (p *DeliveryRetryPolicy) MaxIntervalOrDefault() time.Duration {
	if p == nil || p.MaxIntervalSeconds <= 0 {
		return DefaultDeliveryMaxInterval
	}
	interval := time.Duration(p.MaxIntervalSeconds) * time.Second
	if initial := p.InitialIntervalOrDefault(); interval < initial {
		return initial
	}
	return interval
}

func DeliveryRetryPolicyFromConfig(config map[string]any) *DeliveryRetryPolicy {
	policy := &DeliveryRetryPolicy{}
	configured := false
	if attempts, ok := maputils.IntValue(config, ConfigKeyRetryMaxAttempts); ok && attempts > 0 {
		attempts = min(attempts, MaxDeliveryMaxAttempts)
		policy.MaxAttempts = int32(attempts) //nolint:gosec // clamped to MaxDeliveryMaxAttempts
		configured = true
	}
	if interval, ok := maputils.IntValue(
		config,
		ConfigKeyRetryInitialIntervalSeconds,
	); ok && interval > 0 {
		policy.InitialIntervalSeconds = interval
		configured = true
	}
	if interval, ok := maputils.IntValue(
		config,
		ConfigKeyRetryMaxIntervalSeconds,
	); ok && interval > 0 {
		policy.MaxIntervalSeconds = interval
		configured = true
	}
	if !configured {
		return nil
	}
	return policy
}
