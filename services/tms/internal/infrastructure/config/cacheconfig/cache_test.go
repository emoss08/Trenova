package cacheconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheConfig_GetRedisAddr(t *testing.T) {
	t.Parallel()

	t.Run("joins host and port", func(t *testing.T) {
		t.Parallel()
		c := &Config{Host: "localhost", Port: 6379}
		assert.Equal(t, "localhost:6379", c.GetRedisAddr())
	})

	t.Run("custom host and port", func(t *testing.T) {
		t.Parallel()
		c := &Config{Host: "redis.example.com", Port: 6380}
		assert.Equal(t, "redis.example.com:6380", c.GetRedisAddr())
	})
}
