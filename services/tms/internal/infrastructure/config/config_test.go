package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTemporalConfig_GetNamespace(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalConfig{}
		assert.Equal(t, "default", c.GetNamespace())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalConfig{Namespace: "custom-ns"}
		assert.Equal(t, "custom-ns", c.GetNamespace())
	})
}

func TestTemporalConfig_GetIdentity(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalConfig{}
		assert.Equal(t, "trenova-tms", c.GetIdentity())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalConfig{Identity: "my-service"}
		assert.Equal(t, "my-service", c.GetIdentity())
	})
}

func TestTemporalInterceptorConfig_GetLogLevel(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalInterceptorConfig{}
		assert.Equal(t, "info", c.GetLogLevel())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalInterceptorConfig{LogLevel: "debug"}
		assert.Equal(t, "debug", c.GetLogLevel())
	})
}

func TestTemporalWorkerConfig_GetMaxConcurrentActivities(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{}
		assert.Equal(t, 10, c.GetMaxConcurrentActivities())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{MaxConcurrentActivities: 50}
		assert.Equal(t, 50, c.GetMaxConcurrentActivities())
	})
}

func TestTemporalWorkerConfig_GetMaxConcurrentWorkflows(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{}
		assert.Equal(t, 10, c.GetMaxConcurrentWorkflows())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{MaxConcurrentWorkflows: 25}
		assert.Equal(t, 25, c.GetMaxConcurrentWorkflows())
	})
}

func TestTemporalWorkerConfig_GetMaxActivityPollers(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{}
		assert.Equal(t, 2, c.GetMaxActivityPollers())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{MaxActivityPollers: 8}
		assert.Equal(t, 8, c.GetMaxActivityPollers())
	})
}

func TestTemporalWorkerConfig_GetMaxWorkflowPollers(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{}
		assert.Equal(t, 2, c.GetMaxWorkflowPollers())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{MaxWorkflowPollers: 6}
		assert.Equal(t, 6, c.GetMaxWorkflowPollers())
	})
}

func TestTemporalWorkerConfig_GetWorkerStopTimeout(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{}
		assert.Equal(t, 30*time.Second, c.GetWorkerStopTimeout())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &TemporalWorkerConfig{WorkerStopTimeout: 60 * time.Second}
		assert.Equal(t, 60*time.Second, c.GetWorkerStopTimeout())
	})
}

func TestStorageConfig_GetMaxFileSize(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &StorageConfig{}
		assert.Equal(t, int64(52428800), c.GetMaxFileSize())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &StorageConfig{MaxFileSize: 104857600}
		assert.Equal(t, int64(104857600), c.GetMaxFileSize())
	})
}

func TestStorageConfig_GetPresignedURLExpiry(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &StorageConfig{}
		assert.Equal(t, 15*time.Minute, c.GetPresignedURLExpiry())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &StorageConfig{PresignedURLExpiry: 30 * time.Minute}
		assert.Equal(t, 30*time.Minute, c.GetPresignedURLExpiry())
	})
}

func TestStorageConfig_GetAllowedMIMETypes(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &StorageConfig{}
		result := c.GetAllowedMIMETypes()
		assert.Len(t, result, 12)
		assert.Contains(t, result, "application/pdf")
		assert.Contains(t, result, "image/jpeg")
		assert.Contains(t, result, "text/csv")
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		custom := []string{"image/png", "application/pdf"}
		c := &StorageConfig{AllowedMIMETypes: custom}
		assert.Equal(t, custom, c.GetAllowedMIMETypes())
	})
}

func TestAuditConfig_GetBufferFlushInterval(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{}
		assert.Equal(t, 10*time.Second, c.GetBufferFlushInterval())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{BufferFlushInterval: 20 * time.Second}
		assert.Equal(t, 20*time.Second, c.GetBufferFlushInterval())
	})
}

func TestAuditConfig_GetBatchSize(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{}
		assert.Equal(t, 500, c.GetBatchSize())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{BatchSize: 1000}
		assert.Equal(t, 1000, c.GetBatchSize())
	})
}

func TestAuditConfig_GetMaxEntriesPerFlush(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{}
		assert.Equal(t, 5000, c.GetMaxEntriesPerFlush())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{MaxEntriesPerFlush: 10000}
		assert.Equal(t, 10000, c.GetMaxEntriesPerFlush())
	})
}

func TestAuditConfig_GetDLQRetryInterval(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{}
		assert.Equal(t, 5*time.Minute, c.GetDLQRetryInterval())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{DLQRetryInterval: 10 * time.Minute}
		assert.Equal(t, 10*time.Minute, c.GetDLQRetryInterval())
	})
}

func TestAuditConfig_GetDLQMaxRetries(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{}
		assert.Equal(t, 5, c.GetDLQMaxRetries())
	})

	t.Run("custom", func(t *testing.T) {
		t.Parallel()
		c := &AuditConfig{DLQMaxRetries: 10}
		assert.Equal(t, 10, c.GetDLQMaxRetries())
	})
}

func TestCacheConfig_GetRedisAddr(t *testing.T) {
	t.Parallel()

	t.Run("joins host and port", func(t *testing.T) {
		t.Parallel()
		c := &CacheConfig{Host: "localhost", Port: 6379}
		assert.Equal(t, "localhost:6379", c.GetRedisAddr())
	})

	t.Run("custom host and port", func(t *testing.T) {
		t.Parallel()
		c := &CacheConfig{Host: "redis.example.com", Port: 6380}
		assert.Equal(t, "redis.example.com:6380", c.GetRedisAddr())
	})
}

func TestAppConfig_IsDevelopment(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvDevelopment}
		assert.True(t, c.IsDevelopment())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvProduction}
		assert.False(t, c.IsDevelopment())
	})
}

func TestAppConfig_IsProduction(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvProduction}
		assert.True(t, c.IsProduction())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvDevelopment}
		assert.False(t, c.IsProduction())
	})
}

func TestAppConfig_IsStaging(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvStaging}
		assert.True(t, c.IsStaging())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvDevelopment}
		assert.False(t, c.IsStaging())
	})
}

func TestAppConfig_IsTest(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvTest}
		assert.True(t, c.IsTest())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{Env: EnvDevelopment}
		assert.False(t, c.IsTest())
	})
}

func TestAppConfig_GetProblemTypeBaseURI(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{}
		assert.Equal(t, "https://api.trenova.app/problems/", c.GetProblemTypeBaseURI())
	})

	t.Run("custom with trailing slash", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{ProblemTypeBaseURI: "https://example.com/errors/"}
		assert.Equal(t, "https://example.com/errors/", c.GetProblemTypeBaseURI())
	})

	t.Run("custom without trailing slash", func(t *testing.T) {
		t.Parallel()
		c := &AppConfig{ProblemTypeBaseURI: "https://example.com/errors"}
		assert.Equal(t, "https://example.com/errors/", c.GetProblemTypeBaseURI())
	})
}

func TestConfig_GetCacheConfig(t *testing.T) {
	t.Parallel()

	c := &Config{Cache: CacheConfig{Host: "localhost", Port: 6379}}
	assert.Equal(t, &c.Cache, c.GetCacheConfig())
}

func TestConfig_GetTemporalConfig(t *testing.T) {
	t.Parallel()

	c := &Config{Temporal: TemporalConfig{HostPort: "localhost:7233"}}
	assert.Equal(t, &c.Temporal, c.GetTemporalConfig())
}

func TestConfig_GetMetricsConfig(t *testing.T) {
	t.Parallel()

	c := &Config{Monitoring: MonitoringConfig{Metrics: MetricsConfig{Enabled: true, Port: 9090}}}
	assert.Equal(t, &c.Monitoring.Metrics, c.GetMetricsConfig())
}

func TestConfig_GetStorageConfig(t *testing.T) {
	t.Parallel()

	c := &Config{Storage: StorageConfig{Bucket: "test-bucket"}}
	assert.Equal(t, &c.Storage, c.GetStorageConfig())
}

func TestConfig_GetDSN(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		c := &Config{
			App: AppConfig{Name: "trenova"},
			Database: DatabaseConfig{
				User:    "user",
				Host:    "localhost",
				Port:    5432,
				Name:    "testdb",
				SSLMode: "disable",
			},
		}
		dsn := c.GetDSN("password123")
		assert.Contains(t, dsn, "postgres://user:password123@localhost:5432/testdb")
		assert.Contains(t, dsn, "sslmode=disable")
		assert.Contains(t, dsn, "application_name=trenova")
		assert.Contains(t, dsn, "connect_timeout=10")
	})

	t.Run("special characters in password", func(t *testing.T) {
		t.Parallel()
		c := &Config{
			App: AppConfig{Name: "trenova"},
			Database: DatabaseConfig{
				User:    "admin",
				Host:    "db.example.com",
				Port:    5433,
				Name:    "prod",
				SSLMode: "require",
			},
		}
		dsn := c.GetDSN("p@ss w0rd!")
		assert.Contains(t, dsn, "p%40ss+w0rd%21")
		assert.Contains(t, dsn, "sslmode=require")
	})
}

func TestConfig_GetDSNMasked(t *testing.T) {
	t.Parallel()

	c := &Config{
		Database: DatabaseConfig{
			User:    "user",
			Host:    "localhost",
			Port:    5432,
			Name:    "testdb",
			SSLMode: "disable",
		},
	}
	masked := c.GetDSNMasked()
	assert.Equal(t, "postgres://user:****@localhost:5432/testdb?sslmode=disable", masked)
}

func TestConfig_CorsEnabled(t *testing.T) {
	t.Parallel()

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()
		c := &Config{Server: ServerConfig{CORS: CORSConfig{Enabled: true}}}
		assert.True(t, c.CorsEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		c := &Config{Server: ServerConfig{CORS: CORSConfig{Enabled: false}}}
		assert.False(t, c.CorsEnabled())
	})
}
