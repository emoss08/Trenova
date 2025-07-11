package testutils

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// TestDBWithReplicas creates a test database connection with simulated read replicas
// This is useful for testing read/write separation logic
func TestDBWithReplicas(t *testing.T, numReplicas int) db.Connection {
	ctx := context.Background()

	// * Get the base test database
	testDB := GetTestDB()
	require.NotNil(t, testDB)

	// * Get the underlying bun.DB
	writeDB, err := testDB.DB(ctx)
	require.NoError(t, err)

	// * Create read replicas (in test, they're the same DB)
	readDBs := make([]*bun.DB, numReplicas)
	for i := 0; i < numReplicas; i++ {
		readDBs[i] = writeDB
	}

	// * Return mock connection with tracking
	return NewMockDBConnection(writeDB, readDBs...)
}

// AssertReadOperation verifies that a database operation used a read connection
func AssertReadOperation(t *testing.T, conn db.Connection, operation func() error) {
	mockConn, ok := conn.(*MockDBConnection)
	require.True(t, ok, "Connection must be MockDBConnection for assertion")

	// * Reset counters
	mockConn.ResetCounters()

	// * Execute operation
	err := operation()
	require.NoError(t, err)

	// * Verify read connection was used
	require.Greater(
		t,
		mockConn.GetReadCount(),
		int64(0),
		"Operation should have used read connection",
	)
	require.Equal(
		t,
		int64(0),
		mockConn.GetWriteCount(),
		"Operation should not have used write connection",
	)
}

// AssertWriteOperation verifies that a database operation used a write connection
func AssertWriteOperation(t *testing.T, conn db.Connection, operation func() error) {
	mockConn, ok := conn.(*MockDBConnection)
	require.True(t, ok, "Connection must be MockDBConnection for assertion")

	// * Reset counters
	mockConn.ResetCounters()

	// * Execute operation
	err := operation()
	require.NoError(t, err)

	// * Verify write connection was used
	require.Greater(
		t,
		mockConn.GetWriteCount(),
		int64(0),
		"Operation should have used write connection",
	)
}

// SimulateReplicaFailure simulates read replica failures for testing
func SimulateReplicaFailure(t *testing.T, conn db.Connection, fail bool) {
	mockConn, ok := conn.(*MockDBConnection)
	require.True(t, ok, "Connection must be MockDBConnection for simulation")

	mockConn.SimulateReplicaFailure(fail)
}

// GetOperationCounts returns the read and write operation counts
func GetOperationCounts(t *testing.T, conn db.Connection) (readCount, writeCount int64) {
	mockConn, ok := conn.(*MockDBConnection)
	require.True(t, ok, "Connection must be MockDBConnection to get counts")

	return mockConn.GetReadCount(), mockConn.GetWriteCount()
}
