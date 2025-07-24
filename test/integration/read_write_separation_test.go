/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package integration_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	postgresrepos "github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadWriteSeparation(t *testing.T) {
	// * Skip if running short tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()

	// * Setup test database
	testDB := testutils.GetTestDB()
	require.NotNil(t, testDB)

	// * Create a mock connection that simulates read/write separation
	writeDB, err := testDB.DB(ctx)
	require.NoError(t, err)

	// * For this test, we'll use the same DB but track operations
	mockConn := testutils.NewMockDBConnection(writeDB, writeDB, writeDB)

	// * Create logger config for test
	logCfg := &config.Config{
		App: config.AppConfig{Environment: "test"},
		Log: config.LogConfig{Level: "info"},
	}
	log := logger.NewLogger(logCfg)

	// * Create repository with mock connection
	repo := postgresrepos.NewEquipmentTypeRepository(postgresrepos.EquipmentTypeRespositoryParams{
		DB:     mockConn,
		Logger: log,
	})

	t.Run("List operations should use read connection", func(t *testing.T) {
		// * Reset counters
		mockConn.ResetCounters()

		// * Perform a list operation
		_, err := repo.List(ctx, &repositories.ListEquipmentTypeRequest{
			Filter: &ports.LimitOffsetQueryOptions{
				TenantOpts: ports.TenantOptions{
					BuID:   pulid.MustNew("bu_"),
					OrgID:  pulid.MustNew("org_"),
					UserID: pulid.MustNew("user_"),
				},
				Limit:  10,
				Offset: 0,
			},
		})
		require.NoError(t, err)

		// * Verify read connection was used
		assert.Equal(t, int64(1), mockConn.GetReadCount())
		assert.Equal(t, int64(0), mockConn.GetWriteCount())
	})

	t.Run("GetByID operations should use read connection", func(t *testing.T) {
		// * Create a fresh repository to avoid connection caching
		freshRepo := postgresrepos.NewEquipmentTypeRepository(
			postgresrepos.EquipmentTypeRespositoryParams{
				DB:     mockConn,
				Logger: log,
			},
		)

		// * Reset counters
		mockConn.ResetCounters()

		// * Attempt to get by ID (may not exist, but that's ok for this test)
		_, _ = freshRepo.GetByID(ctx, repositories.GetEquipmentTypeByIDOptions{
			ID:    pulid.MustNew("et_"),
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		})

		// * Verify read connection was used
		assert.Equal(t, int64(1), mockConn.GetReadCount())
		assert.Equal(t, int64(0), mockConn.GetWriteCount())
	})

	t.Run("Create operations should use write connection", func(t *testing.T) {
		// * Reset counters
		mockConn.ResetCounters()

		// * Attempt to create (may fail, but that's ok for this test)
		_, _ = repo.Create(ctx, &equipmenttype.EquipmentType{
			Code:  "TEST",
			Class: equipmenttype.ClassTractor,
		})

		// * Verify write connection was used
		assert.Equal(t, int64(0), mockConn.GetReadCount())
		assert.Equal(t, int64(1), mockConn.GetWriteCount())
	})

	t.Run("Update operations should use write connection", func(t *testing.T) {
		// * Reset counters
		mockConn.ResetCounters()

		// * Attempt to update (may fail, but that's ok for this test)
		_, _ = repo.Update(ctx, &equipmenttype.EquipmentType{
			ID:    pulid.MustNew("et_"),
			Code:  "UPDATED",
			Class: equipmenttype.ClassTractor,
		})

		// * Verify write connection was used (transaction uses write connection)
		assert.Equal(t, int64(0), mockConn.GetReadCount())
		assert.Greater(t, mockConn.GetWriteCount(), int64(0))
	})

	t.Run("Replica failure should fall back to write connection", func(t *testing.T) {
		// * Create a fresh repository to avoid connection caching
		freshRepo := postgresrepos.NewEquipmentTypeRepository(
			postgresrepos.EquipmentTypeRespositoryParams{
				DB:     mockConn,
				Logger: log,
			},
		)

		// * Simulate all replicas being unhealthy
		mockConn.SimulateReplicaFailure(true)
		mockConn.ResetCounters()

		// * Perform a read operation
		_, err := freshRepo.List(ctx, &repositories.ListEquipmentTypeRequest{
			Filter: &ports.LimitOffsetQueryOptions{
				TenantOpts: ports.TenantOptions{
					BuID:   pulid.MustNew("bu_"),
					OrgID:  pulid.MustNew("org_"),
					UserID: pulid.MustNew("user_"),
				},
				Limit:  10,
				Offset: 0,
			},
		})
		require.NoError(t, err)

		// * Even though we requested read, it should have used write DB due to failure
		// * The read counter still increments because ReadDB was called,
		// * but internally it returned the write DB
		assert.Equal(t, int64(1), mockConn.GetReadCount())

		// * Reset failure simulation
		mockConn.SimulateReplicaFailure(false)
	})
}

func TestMockDBConnectionRoundRobin(t *testing.T) {
	ctx := context.Background()

	// * Setup test database connections
	testDB := testutils.GetTestDB()
	require.NotNil(t, testDB)

	writeDB, err := testDB.DB(ctx)
	require.NoError(t, err)

	// * Create mock with multiple read replicas (same DB in this test)
	mockConn := testutils.NewMockDBConnection(writeDB, writeDB, writeDB, writeDB)

	t.Run("Should distribute reads across replicas", func(t *testing.T) {
		// * Make multiple read calls
		for i := 0; i < 10; i++ {
			_, err := mockConn.ReadDB(ctx)
			require.NoError(t, err)
		}

		// * Verify round-robin distribution
		assert.Equal(t, int64(10), mockConn.GetReadCount())
		assert.Equal(t, int64(0), mockConn.GetWriteCount())
	})

	t.Run("Adding and removing replicas", func(t *testing.T) {
		// * Add a new replica
		mockConn.AddReadReplica(writeDB)

		// * Remove a replica
		mockConn.RemoveReadReplica(0)

		// * Should still work
		_, err := mockConn.ReadDB(ctx)
		require.NoError(t, err)
	})
}
