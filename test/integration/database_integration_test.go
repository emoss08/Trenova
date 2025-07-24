/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseReadWriteSeparation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// * Use the test database connection
	testDB := testutils.GetTestDB()
	require.NotNil(t, testDB)

	t.Run("Connection initialization", func(t *testing.T) {
		// Test primary connection
		primaryDB, err := testDB.WriteDB(ctx)
		require.NoError(t, err)
		require.NotNil(t, primaryDB)

		// Test read connection
		readDB, err := testDB.ReadDB(ctx)
		require.NoError(t, err)
		require.NotNil(t, readDB)
	})

	t.Run("Connection info", func(t *testing.T) {
		info, err := testDB.ConnectionInfo()
		require.NoError(t, err)
		assert.NotEmpty(t, info.Host)
		assert.NotZero(t, info.Port)
		assert.NotEmpty(t, info.Database)
	})

	t.Run("Multiple read requests", func(t *testing.T) {
		// Should handle multiple concurrent read requests
		results := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, err := testDB.ReadDB(ctx)
				results <- err
			}()
		}

		// Collect results
		for i := 0; i < 10; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})
}

func TestConnectionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	ctx := context.Background()

	// * Use the test database connection
	conn := testutils.GetTestDB()
	require.NotNil(t, conn)

	// Initialize connection
	_, err := conn.DB(ctx)
	require.NoError(t, err)

	t.Run("Connection retrieval performance", func(t *testing.T) {
		// Warm up
		for i := 0; i < 100; i++ {
			conn.ReadDB(ctx)
			conn.WriteDB(ctx)
		}

		// Measure performance
		iterations := 10000

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := conn.ReadDB(ctx)
			require.NoError(t, err)
		}
		readDuration := time.Since(start)
		avgRead := readDuration / time.Duration(iterations)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			_, err := conn.WriteDB(ctx)
			require.NoError(t, err)
		}
		writeDuration := time.Since(start)
		avgWrite := writeDuration / time.Duration(iterations)

		// Performance should be excellent
		assert.Less(t, avgRead, time.Microsecond,
			"ReadDB average duration %v exceeds 1μs", avgRead)
		assert.Less(t, avgWrite, time.Microsecond,
			"WriteDB average duration %v exceeds 1μs", avgWrite)

		t.Logf("Performance results:")
		t.Logf("  ReadDB avg: %v", avgRead)
		t.Logf("  WriteDB avg: %v", avgWrite)
	})
}
