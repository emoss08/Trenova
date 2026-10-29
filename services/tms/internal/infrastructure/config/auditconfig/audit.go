package auditconfig

import (
	"time"
)

type Config struct {
	BufferFlushInterval time.Duration `mapstructure:"bufferFlushInterval"`
	BatchSize           int           `mapstructure:"batchSize"           validate:"omitempty,min=1,max=5000"`
	MaxEntriesPerFlush  int           `mapstructure:"maxEntriesPerFlush"  validate:"omitempty,min=100,max=50000"`
	DLQRetryInterval    time.Duration `mapstructure:"dlqRetryInterval"`
	DLQMaxRetries       int           `mapstructure:"dlqMaxRetries"       validate:"omitempty,min=1,max=20"`
}

func (c *Config) GetBufferFlushInterval() time.Duration {
	if c.BufferFlushInterval == 0 {
		return time.Minute
	}

	return c.BufferFlushInterval
}

func (c *Config) GetBatchSize() int {
	if c.BatchSize == 0 {
		return 500
	}

	return c.BatchSize
}

func (c *Config) GetMaxEntriesPerFlush() int {
	if c.MaxEntriesPerFlush == 0 {
		return 5000
	}

	return c.MaxEntriesPerFlush
}

func (c *Config) GetDLQRetryInterval() time.Duration {
	if c.DLQRetryInterval == 0 {
		return 5 * time.Minute
	}

	return c.DLQRetryInterval
}

func (c *Config) GetDLQMaxRetries() int {
	if c.DLQMaxRetries == 0 {
		return 5
	}

	return c.DLQMaxRetries
}
