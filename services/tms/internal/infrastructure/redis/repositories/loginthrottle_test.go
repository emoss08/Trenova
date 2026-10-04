package repositories

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoginThrottleKeysNeverHoldTheAddress(t *testing.T) {
	t.Parallel()

	store := &loginThrottleStore{}
	key := store.accountFailuresKey(" Dana@Example.com ")

	assert.NotContains(t, strings.ToLower(key), "dana")
	assert.Equal(t, key, store.accountFailuresKey("dana@example.com"))
	assert.NotEqual(t, key, store.accountLockKey("dana@example.com"))
	assert.True(t, strings.HasPrefix(key, loginThrottlePrefix))
}

func TestLoginThrottleIPKey(t *testing.T) {
	t.Parallel()

	store := &loginThrottleStore{}
	assert.Equal(t, loginThrottlePrefix+":ip:203.0.113.9", store.ipFailuresKey(" 203.0.113.9 "))
	assert.Equal(t, loginThrottlePrefix+":ip:unknown", store.ipFailuresKey(""))
}

func TestPositiveOr(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Minute, positiveOr(-1, time.Minute))
	assert.Equal(t, time.Second, positiveOr(time.Second, time.Minute))
}
