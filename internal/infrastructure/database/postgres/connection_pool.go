// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package postgres

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/pkg/metrics"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// ConnectionPool manages database connections with performance optimizations
type ConnectionPool struct {
	primary         *bun.DB
	replicas        []*readReplica
	healthyReplicas atomic.Value // Cached healthy replicas for fast access
	currentIdx      uint64
	mu              sync.RWMutex
	log             *zerolog.Logger

	// Performance optimization: cache the decision
	readFromPrimary atomic.Bool
	lastHealthCheck time.Time
}

// HealthyReplicaList is a cached list of healthy replicas
type HealthyReplicaList struct {
	replicas []*readReplica
	count    int
}

// NewConnectionPool creates a new optimized connection pool
func NewConnectionPool(primary *bun.DB, log *zerolog.Logger) *ConnectionPool {
	pool := &ConnectionPool{
		primary: primary,
		log:     log,
	}

	// Initialize with empty healthy list
	pool.healthyReplicas.Store(&HealthyReplicaList{
		replicas: make([]*readReplica, 0),
		count:    0,
	})

	return pool
}

// GetReadConnection returns a read connection with minimal overhead
func (cp *ConnectionPool) GetReadConnection() (*bun.DB, string) {
	start := time.Now()
	defer func() {
		metrics.RecordDatabaseOperation("get_read_connection", "pool", time.Since(start))
	}()

	// Fast path: check if we should use primary
	if cp.readFromPrimary.Load() {
		metrics.RecordReadWriteDistribution("primary", "read")
		return cp.primary, "primary"
	}

	// Get cached healthy replicas without lock
	healthyList, _ := cp.healthyReplicas.Load().(*HealthyReplicaList)
	if healthyList.count == 0 {
		metrics.RecordReadWriteDistribution("primary", "read")
		return cp.primary, "primary"
	}

	// Simple round-robin without modulo (faster)
	idx := atomic.AddUint64(&cp.currentIdx, 1)
	replica := healthyList.replicas[idx%uint64(len(healthyList.replicas))]

	metrics.RecordReadWriteDistribution(replica.name, "read")
	return replica.db, replica.name
}

// GetWriteConnection returns the primary connection
func (cp *ConnectionPool) GetWriteConnection() (*bun.DB, string) {
	metrics.RecordReadWriteDistribution("primary", "write")
	return cp.primary, "primary"
}

// AddReplica adds a read replica to the pool
func (cp *ConnectionPool) AddReplica(replica *readReplica) {
	cp.mu.Lock()
	cp.replicas = append(cp.replicas, replica)
	cp.mu.Unlock()

	// Update healthy list
	cp.updateHealthyReplicas()
}

// updateHealthyReplicas updates the cached list of healthy replicas
func (cp *ConnectionPool) updateHealthyReplicas() {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	healthy := make([]*readReplica, 0, len(cp.replicas))
	for _, r := range cp.replicas {
		r.mu.RLock()
		if r.healthy {
			healthy = append(healthy, r)
		}
		r.mu.RUnlock()
	}

	// Update atomic value with new list
	cp.healthyReplicas.Store(&HealthyReplicaList{
		replicas: healthy,
		count:    len(healthy),
	})

	// Update fast path flag
	cp.readFromPrimary.Store(len(healthy) == 0)

	cp.log.Debug().
		Int("healthy_count", len(healthy)).
		Int("total_count", len(cp.replicas)).
		Msg("updated healthy replica list")
}

// HealthCheck performs health checks on all replicas
func (cp *ConnectionPool) HealthCheck(ctx context.Context, lagThreshold time.Duration) {
	cp.mu.RLock()
	replicas := make([]*readReplica, len(cp.replicas))
	copy(replicas, cp.replicas)
	cp.mu.RUnlock()

	// Check replicas in parallel for speed
	var wg sync.WaitGroup
	for _, replica := range replicas {
		wg.Add(1)
		go func(r *readReplica) {
			defer wg.Done()
			cp.checkReplicaHealth(ctx, r, lagThreshold)
		}(replica)
	}
	wg.Wait()

	// Update cached healthy list
	cp.updateHealthyReplicas()
	cp.lastHealthCheck = time.Now()
}

// checkReplicaHealth checks a single replica's health
func (cp *ConnectionPool) checkReplicaHealth(
	ctx context.Context,
	replica *readReplica,
	lagThreshold time.Duration,
) {
	start := time.Now()
	defer func() {
		metrics.RecordDatabaseOperation("health_check", replica.name, time.Since(start))
	}()

	// Use a short timeout for health checks
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Ping check
	if replica.db == nil {
		cp.log.Warn().
			Str("replica", replica.name).
			Msg("replica has nil db connection")
		replica.mu.Lock()
		replica.healthy = false
		replica.mu.Unlock()
		return
	}

	// Get underlying sql.DB to ping
	sqlDB := replica.db.DB
	err := sqlDB.PingContext(checkCtx)

	replica.mu.Lock()
	oldHealthy := replica.healthy

	if err != nil {
		replica.healthy = false
		if oldHealthy {
			cp.log.Error().
				Err(err).
				Str("replica", replica.name).
				Msg("replica failed health check")
		}
		metrics.UpdateReplicaHealth(replica.name, false)
		replica.mu.Unlock()
		return
	}

	// Check replication lag if threshold is set
	if lagThreshold > 0 { //nolint:nestif // this is necessary
		lag, lErr := cp.getReplicationLag(checkCtx, replica)
		if lErr != nil {
			cp.log.Error().
				Err(lErr).
				Str("replica", replica.name).
				Msg("failed to check replication lag")
		} else {
			metrics.UpdateReplicationLag(replica.name, lag.Seconds())

			if lag > lagThreshold {
				replica.healthy = false
				if oldHealthy {
					cp.log.Warn().
						Str("replica", replica.name).
						Dur("lag", lag).
						Dur("threshold", lagThreshold).
						Msg("replica lag exceeds threshold")
				}
				metrics.UpdateReplicaHealth(replica.name, false)
				replica.mu.Unlock()
				return
			}
		}
	}

	// Mark as healthy
	replica.healthy = true
	if !oldHealthy {
		cp.log.Info().
			Str("replica", replica.name).
			Msg("replica recovered and is now healthy")
	}
	metrics.UpdateReplicaHealth(replica.name, true)
	replica.lastCheck = time.Now()
	replica.mu.Unlock()
}

// getReplicationLag efficiently checks replication lag
func (cp *ConnectionPool) getReplicationLag(
	ctx context.Context,
	replica *readReplica,
) (time.Duration, error) {
	var lagSeconds float64

	// Use a prepared query for better performance
	query := `SELECT EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - pg_last_xact_replay_timestamp()))::FLOAT AS lag_seconds`

	err := replica.db.QueryRowContext(ctx, query).Scan(&lagSeconds)
	if err != nil {
		return 0, err
	}

	// Handle NULL case (no replication lag info)
	if lagSeconds < 0 {
		return 0, nil
	}

	return time.Duration(lagSeconds * float64(time.Second)), nil
}

// GetPoolStats returns connection pool statistics
func (cp *ConnectionPool) GetPoolStats() {
	// Primary connection stats
	if cp.primary != nil && cp.primary.DB != nil {
		stats := cp.primary.Stats()

		metrics.UpdateConnectionPoolStats("primary",
			intutils.SafeInt32(stats.MaxOpenConnections),
			intutils.SafeInt32(stats.Idle),
			intutils.SafeInt32(stats.InUse))
	}

	// Replica stats
	cp.mu.RLock()
	for _, replica := range cp.replicas {
		if replica.db != nil && replica.db.DB != nil {
			stats := replica.db.Stats()

			metrics.UpdateConnectionPoolStats(replica.name,
				intutils.SafeInt32(stats.MaxOpenConnections),
				intutils.SafeInt32(stats.Idle),
				intutils.SafeInt32(stats.InUse))
		}
	}
	cp.mu.RUnlock()
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Close replicas
	for _, replica := range cp.replicas {
		if replica.db != nil {
			if err := replica.db.Close(); err != nil {
				cp.log.Error().
					Err(err).
					Str("replica", replica.name).
					Msg("error closing replica connection")
			}
		}
		if replica.pool != nil {
			replica.pool.Close()
		}
	}

	return nil
}
