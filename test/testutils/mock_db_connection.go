// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package testutils

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/uptrace/bun"
)

// MockDBConnection simulates read/write separation for testing
type MockDBConnection struct {
	writeDB      *bun.DB
	readDBs      []*bun.DB
	currentRead  int32
	readCount    int64
	writeCount   int64
	mu           sync.RWMutex
	forceFailure bool
}

// NewMockDBConnection creates a new mock database connection
// This can be used to test read/write separation logic
func NewMockDBConnection(writeDB *bun.DB, readDBs ...*bun.DB) *MockDBConnection {
	return &MockDBConnection{
		writeDB: writeDB,
		readDBs: readDBs,
	}
}

// DB returns the write database (for backward compatibility)
func (m *MockDBConnection) DB(ctx context.Context) (*bun.DB, error) {
	return m.WriteDB(ctx)
}

// ReadDB returns a read database using round-robin selection
func (m *MockDBConnection) ReadDB(ctx context.Context) (*bun.DB, error) {
	atomic.AddInt64(&m.readCount, 1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.forceFailure {
		// * Simulate all replicas being unhealthy, fall back to write DB
		return m.writeDB, nil
	}

	if len(m.readDBs) == 0 {
		// * No read replicas configured, use write DB
		return m.writeDB, nil
	}

	// * Simple round-robin selection
	idx := atomic.AddInt32(&m.currentRead, 1)
	selectedDB := m.readDBs[int(idx-1)%len(m.readDBs)]

	return selectedDB, nil
}

// WriteDB returns the write database
func (m *MockDBConnection) WriteDB(ctx context.Context) (*bun.DB, error) {
	atomic.AddInt64(&m.writeCount, 1)
	return m.writeDB, nil
}

// ConnectionInfo returns mock connection information
func (m *MockDBConnection) ConnectionInfo() (*db.ConnectionInfo, error) {
	return &db.ConnectionInfo{
		Host:     "localhost",
		Port:     5432,
		Database: "test_db",
		Username: "test_user",
		Password: "test_password",
		SSLMode:  "disable",
	}, nil
}

// SQLDB returns the underlying sql.DB
func (m *MockDBConnection) SQLDB(ctx context.Context) (*sql.DB, error) {
	return m.writeDB.DB, nil
}

// Close closes all database connections
func (m *MockDBConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// * Close write DB
	if m.writeDB != nil {
		if err := m.writeDB.Close(); err != nil {
			return err
		}
	}

	// * Close all read DBs
	for _, readDB := range m.readDBs {
		if readDB != nil {
			if err := readDB.Close(); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetReadCount returns the number of read operations
func (m *MockDBConnection) GetReadCount() int64 {
	return atomic.LoadInt64(&m.readCount)
}

// GetWriteCount returns the number of write operations
func (m *MockDBConnection) GetWriteCount() int64 {
	return atomic.LoadInt64(&m.writeCount)
}

// ResetCounters resets the read/write counters
func (m *MockDBConnection) ResetCounters() {
	atomic.StoreInt64(&m.readCount, 0)
	atomic.StoreInt64(&m.writeCount, 0)
}

// SimulateReplicaFailure simulates all read replicas being unhealthy
func (m *MockDBConnection) SimulateReplicaFailure(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceFailure = fail
}

// AddReadReplica adds a new read replica to the pool
func (m *MockDBConnection) AddReadReplica(db *bun.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readDBs = append(m.readDBs, db)
}

// RemoveReadReplica removes a read replica from the pool
func (m *MockDBConnection) RemoveReadReplica(index int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index >= 0 && index < len(m.readDBs) {
		m.readDBs = append(m.readDBs[:index], m.readDBs[index+1:]...)
	}
}
