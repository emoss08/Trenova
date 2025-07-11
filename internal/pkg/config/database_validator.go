package config

import (
	"fmt"
	"net"

	"github.com/rotisserie/eris"
)

// ValidateDatabaseConfig validates database configuration including replicas
func ValidateDatabaseConfig(cfg *DatabaseConfig) error {
	// Validate primary database
	if err := validateDatabaseConnection(cfg.Host, cfg.Port, "primary"); err != nil {
		return err
	}

	// Validate basic settings
	if cfg.MaxConnections <= 0 {
		return eris.New("maxConnections must be greater than 0")
	}

	if cfg.MaxIdleConns < 0 {
		return eris.New("maxIdleConns cannot be negative")
	}

	if cfg.MaxIdleConns > cfg.MaxConnections {
		return eris.New("maxIdleConns cannot exceed maxConnections")
	}

	// Validate read/write separation if enabled
	if cfg.EnableReadWriteSeparation {
		if len(cfg.ReadReplicas) == 0 {
			return eris.New("enableReadWriteSeparation is true but no readReplicas configured")
		}

		if cfg.ReplicaLagThreshold < 0 {
			return eris.New("replicaLagThreshold cannot be negative")
		}

		// Validate each replica
		replicaNames := make(map[string]bool)
		totalWeight := 0

		for i, replica := range cfg.ReadReplicas {
			// Check for duplicate names
			if replica.Name == "" {
				return eris.Errorf("readReplicas[%d]: name is required", i)
			}

			if replicaNames[replica.Name] {
				return eris.Errorf("duplicate replica name: %s", replica.Name)
			}
			replicaNames[replica.Name] = true

			// Validate connection
			if err := validateDatabaseConnection(replica.Host, replica.Port, replica.Name); err != nil {
				return eris.Wrapf(err, "readReplicas[%d]", i)
			}

			// Validate weight
			if replica.Weight < 0 {
				return eris.Errorf("readReplicas[%d]: weight cannot be negative", i)
			}

			if replica.Weight == 0 {
				replica.Weight = 1 // Default weight
			}

			totalWeight += replica.Weight

			// Validate connection pool settings
			if replica.MaxConnections > 0 && replica.MaxIdleConns > replica.MaxConnections {
				return eris.Errorf("readReplicas[%d]: maxIdleConns cannot exceed maxConnections", i)
			}
		}

		// Warn if weights are unbalanced
		if totalWeight > 0 && len(cfg.ReadReplicas) > 1 {
			avgWeight := totalWeight / len(cfg.ReadReplicas)
			for _, replica := range cfg.ReadReplicas {
				if replica.Weight > avgWeight*3 {
					// This is just a warning in logs, not an error
					fmt.Printf(
						"WARNING: Replica %s has weight %d, which is significantly higher than average %d\n",
						replica.Name,
						replica.Weight,
						avgWeight,
					)
				}
			}
		}
	}

	return nil
}

// validateDatabaseConnection validates host and port
func validateDatabaseConnection(host string, port int, name string) error {
	if host == "" {
		return eris.Errorf("%s: host is required", name)
	}

	if port <= 0 || port > 65535 {
		return eris.Errorf("%s: invalid port %d", name, port)
	}

	// Validate host format (basic check)
	if _, err := net.LookupHost(host); err != nil {
		// Check if it's a valid IP
		if net.ParseIP(host) == nil {
			// For local development, allow localhost variants
			if host != "localhost" && host != "db" && host != "host.docker.internal" {
				return eris.Errorf("%s: invalid host %s", name, host)
			}
		}
	}

	return nil
}

// SuggestOptimalSettings suggests optimal configuration based on resources
func SuggestOptimalSettings(totalMemoryMB int, cpuCount int) DatabaseConfig {
	// PostgreSQL tuning recommendations
	// These values would need to be set in postgresql.conf:
	// - Shared buffers: 25% of RAM (totalMemoryMB / 4)
	// - Effective cache size: 50-75% of RAM ((totalMemoryMB * 3) / 4)
	// - Work mem: RAM / (max_connections * 3)

	// Connection pool settings
	maxConnections := cpuCount * 25
	if maxConnections > 200 {
		maxConnections = 200
	}

	maxIdleConns := maxConnections / 2

	return DatabaseConfig{
		MaxConnections:  maxConnections,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: 3600, // 1 hour
		ConnMaxIdleTime: 900,  // 15 minutes

		// If using replicas
		EnableReadWriteSeparation: cpuCount >= 4, // Enable for 4+ CPUs
		ReplicaLagThreshold:       10,            // 10 seconds is reasonable

		// These would need to be set in postgresql.conf
		// Including here for reference
		//SharedBuffers: fmt.Sprintf("%dMB", sharedBuffersMB),
		//EffectiveCacheSize: fmt.Sprintf("%dMB", effectiveCacheSizeMB),
		//WorkMem: fmt.Sprintf("%dMB", workMemMB),
	}
}

// GetConnectionString builds a safe connection string (without password)
func GetSafeConnectionString(cfg *DatabaseConfig) string {
	return fmt.Sprintf("postgresql://%s:****@%s:%d/%s?sslmode=%s",
		cfg.Username,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)
}

// GetReplicaConnectionString builds a connection string for a replica
func GetReplicaConnectionString(cfg *DatabaseConfig, replica *ReadReplicaConfig) string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Username,
		cfg.Password,
		replica.Host,
		replica.Port,
		cfg.Database,
		cfg.SSLMode,
	)
}

// EstimateConnectionPoolMemory estimates memory usage for connection pools
func EstimateConnectionPoolMemory(cfg *DatabaseConfig) int {
	// PostgreSQL connection memory estimation
	// Each connection uses approximately 10MB
	connectionMemoryMB := 10

	primaryMemory := cfg.MaxConnections * connectionMemoryMB

	replicaMemory := 0
	for _, replica := range cfg.ReadReplicas {
		maxConns := replica.MaxConnections
		if maxConns == 0 {
			maxConns = cfg.MaxConnections
		}
		replicaMemory += maxConns * connectionMemoryMB
	}

	return primaryMemory + replicaMemory
}
