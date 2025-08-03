/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// DatabaseConnectionsTotal tracks total database connections by type
	DatabaseConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trenova_database_connections_total",
			Help: "Total number of database connection attempts",
		},
		[]string{"connection_type", "status"},
	)

	// DatabaseOperationsTotal tracks total database operations
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trenova_database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation_type", "connection_type"},
	)

	// DatabaseOperationDuration tracks operation duration
	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trenova_database_operation_duration_seconds",
			Help:    "Database operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation_type", "connection_type"},
	)

	// DatabaseReplicaHealth tracks replica health status
	DatabaseReplicaHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trenova_database_replica_health",
			Help: "Database replica health status (1 = healthy, 0 = unhealthy)",
		},
		[]string{"replica_name"},
	)

	// DatabaseReplicationLag tracks replication lag
	DatabaseReplicationLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trenova_database_replication_lag_seconds",
			Help: "Database replication lag in seconds",
		},
		[]string{"replica_name"},
	)

	// DatabaseConnectionPoolStats tracks connection pool statistics
	DatabaseConnectionPoolStats = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trenova_database_connection_pool_stats",
			Help: "Database connection pool statistics",
		},
		[]string{"connection_type", "stat_type"},
	)

	// ReadWriteDistribution tracks the distribution of read/write operations
	ReadWriteDistribution = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trenova_database_read_write_distribution_total",
			Help: "Distribution of read and write operations across connections",
		},
		[]string{"connection_name", "operation_type"},
	)
)

// RecordDatabaseOperation records a database operation with timing
func RecordDatabaseOperation(operationType, connectionType string, duration time.Duration) {
	DatabaseOperationsTotal.WithLabelValues(operationType, connectionType).Inc()
	DatabaseOperationDuration.WithLabelValues(operationType, connectionType).
		Observe(duration.Seconds())
}

// RecordConnectionAttempt records a connection attempt
func RecordConnectionAttempt(connectionType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	DatabaseConnectionsTotal.WithLabelValues(connectionType, status).Inc()
}

// UpdateReplicaHealth updates the health status of a replica
func UpdateReplicaHealth(replicaName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	DatabaseReplicaHealth.WithLabelValues(replicaName).Set(value)
}

// UpdateReplicationLag updates the replication lag for a replica
func UpdateReplicationLag(replicaName string, lagSeconds float64) {
	DatabaseReplicationLag.WithLabelValues(replicaName).Set(lagSeconds)
}

// UpdateConnectionPoolStats updates connection pool statistics
func UpdateConnectionPoolStats(connectionType string, totalConns, idleConns, activeConns int32) {
	DatabaseConnectionPoolStats.WithLabelValues(connectionType, "total").Set(float64(totalConns))
	DatabaseConnectionPoolStats.WithLabelValues(connectionType, "idle").Set(float64(idleConns))
	DatabaseConnectionPoolStats.WithLabelValues(connectionType, "active").Set(float64(activeConns))
}

// RecordReadWriteDistribution records which connection handled an operation
func RecordReadWriteDistribution(connectionName, operationType string) {
	ReadWriteDistribution.WithLabelValues(connectionName, operationType).Inc()
}
