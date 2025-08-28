/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package config

import "time"

// DockerConfig holds configuration for Docker service
type DockerConfig struct {
	// StatsInterval is the interval for streaming container stats
	StatsInterval time.Duration `env:"DOCKER_STATS_INTERVAL" envDefault:"2s"`

	// DiskUsageTimeout is the timeout for disk usage calculation
	DiskUsageTimeout time.Duration `env:"DOCKER_DISK_USAGE_TIMEOUT" envDefault:"10s"`

	// DiskUsageCacheTTL is the cache TTL for disk usage
	DiskUsageCacheTTL time.Duration `env:"DOCKER_DISK_USAGE_CACHE_TTL" envDefault:"60s"`

	// SystemInfoCacheTTL is the cache TTL for system info
	SystemInfoCacheTTL time.Duration `env:"DOCKER_SYSTEM_INFO_CACHE_TTL" envDefault:"30s"`

	// EnableAuditLog enables audit logging for Docker operations
	EnableAuditLog bool `env:"DOCKER_ENABLE_AUDIT_LOG" envDefault:"true"`

	// EnablePermissionChecks enables permission checks for Docker operations
	EnablePermissionChecks bool `env:"DOCKER_ENABLE_PERMISSION_CHECKS" envDefault:"true"`
}
