package auditconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuditConfig_GetBufferFlushInterval(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, time.Minute, c.GetBufferFlushInterval())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{BufferFlushInterval: 20 * time.Second}
		assert.Equal(t, 20*time.Second, c.GetBufferFlushInterval())
	})
}

func TestAuditConfig_GetBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, 500, c.GetBatchSize())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{BatchSize: 1000}
		assert.Equal(t, 1000, c.GetBatchSize())
	})
}

func TestAuditConfig_GetMaxEntriesPerFlush(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, 5000, c.GetMaxEntriesPerFlush())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{MaxEntriesPerFlush: 10000}
		assert.Equal(t, 10000, c.GetMaxEntriesPerFlush())
	})
}

func TestAuditConfig_GetDLQRetryInterval(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, 5*time.Minute, c.GetDLQRetryInterval())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{DLQRetryInterval: 10 * time.Minute}
		assert.Equal(t, 10*time.Minute, c.GetDLQRetryInterval())
	})
}

func TestAuditConfig_GetDLQMaxRetries(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &Config{}
		assert.Equal(t, 5, c.GetDLQMaxRetries())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &Config{DLQMaxRetries: 10}
		assert.Equal(t, 10, c.GetDLQMaxRetries())
	})
}
