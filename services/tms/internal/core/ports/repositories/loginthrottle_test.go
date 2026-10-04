package repositories_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/stretchr/testify/assert"
)

func TestAccountLockForBacksOffExponentially(t *testing.T) {
	t.Parallel()

	policy := repositories.DefaultLoginThrottlePolicy()

	assert.Zero(t, policy.AccountLockFor(0))
	assert.Zero(t, policy.AccountLockFor(4))
	assert.Equal(t, 30*time.Second, policy.AccountLockFor(5))
	assert.Equal(t, time.Minute, policy.AccountLockFor(6))
	assert.Equal(t, 2*time.Minute, policy.AccountLockFor(7))
	assert.Equal(t, 8*time.Minute, policy.AccountLockFor(9))
	assert.Equal(t, 15*time.Minute, policy.AccountLockFor(10))
	assert.Equal(t, 15*time.Minute, policy.AccountLockFor(1_000))
}

func TestAccountLockForDisabledPolicy(t *testing.T) {
	t.Parallel()

	assert.Zero(t, repositories.LoginThrottlePolicy{}.AccountLockFor(100))
}
