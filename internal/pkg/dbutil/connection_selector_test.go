// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package dbutil_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/pkg/dbutil"
	"github.com/emoss08/trenova/test/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestConnectionSelector(t *testing.T) {
	ctx := context.Background()

	// * Create a test connection with mock replicas
	conn := testutils.TestDBWithReplicas(t, 3)
	selector := dbutil.NewConnectionSelector(conn)

	t.Run("GetDB with ReadOperation", func(t *testing.T) {
		testutils.AssertReadOperation(t, conn, func() error {
			_, err := selector.GetDB(ctx, dbutil.ReadOperation)
			return err
		})
	})

	t.Run("GetDB with WriteOperation", func(t *testing.T) {
		testutils.AssertWriteOperation(t, conn, func() error {
			_, err := selector.GetDB(ctx, dbutil.WriteOperation)
			return err
		})
	})

	t.Run("GetReadDB", func(t *testing.T) {
		testutils.AssertReadOperation(t, conn, func() error {
			_, err := selector.Read(ctx)
			return err
		})
	})

	t.Run("GetWriteDB", func(t *testing.T) {
		testutils.AssertWriteOperation(t, conn, func() error {
			_, err := selector.Write(ctx)
			return err
		})
	})
}

func TestInferOperationType(t *testing.T) {
	tests := []struct {
		name         string
		methodName   string
		expectedType dbutil.OperationType
	}{
		// * Read operations
		{"GetByID", "GetByID", dbutil.ReadOperation},
		{"GetUser", "GetUser", dbutil.ReadOperation},
		{"List", "List", dbutil.ReadOperation},
		{"ListUsers", "ListUsers", dbutil.ReadOperation},
		{"Find", "Find", dbutil.ReadOperation},
		{"FindByEmail", "FindByEmail", dbutil.ReadOperation},
		{"Search", "Search", dbutil.ReadOperation},
		{"SearchUsers", "SearchUsers", dbutil.ReadOperation},
		{"Query", "Query", dbutil.ReadOperation},
		{"QueryByStatus", "QueryByStatus", dbutil.ReadOperation},
		{"Select", "Select", dbutil.ReadOperation},
		{"SelectActive", "SelectActive", dbutil.ReadOperation},
		{"Count", "Count", dbutil.ReadOperation},
		{"CountByType", "CountByType", dbutil.ReadOperation},
		{"Exists", "Exists", dbutil.ReadOperation},
		{"ExistsByEmail", "ExistsByEmail", dbutil.ReadOperation},
		{"Has", "Has", dbutil.ReadOperation},
		{"HasPermission", "HasPermission", dbutil.ReadOperation},
		{"Is", "Is", dbutil.ReadOperation},
		{"IsActive", "IsActive", dbutil.ReadOperation},
		{"Check", "Check", dbutil.ReadOperation},
		{"CheckStatus", "CheckStatus", dbutil.ReadOperation},
		{"Fetch", "Fetch", dbutil.ReadOperation},
		{"FetchLatest", "FetchLatest", dbutil.ReadOperation},

		// * Write operations (default)
		{"Create", "Create", dbutil.WriteOperation},
		{"Update", "Update", dbutil.WriteOperation},
		{"Delete", "Delete", dbutil.WriteOperation},
		{"Insert", "Insert", dbutil.WriteOperation},
		{"Save", "Save", dbutil.WriteOperation},
		{"Remove", "Remove", dbutil.WriteOperation},
		{"Set", "Set", dbutil.WriteOperation},
		{"Add", "Add", dbutil.WriteOperation},
		{"Process", "Process", dbutil.WriteOperation},
		{"Execute", "Execute", dbutil.WriteOperation},
		{"DoSomething", "DoSomething", dbutil.WriteOperation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dbutil.InferOperationType(tt.methodName)
			assert.Equal(t, tt.expectedType, result,
				"Method %s should be inferred as %v", tt.methodName, tt.expectedType)
		})
	}
}

func TestTransactionHelper(t *testing.T) {
	ctx := context.Background()

	// * Create a test connection
	conn := testutils.TestDBWithReplicas(t, 2)
	txHelper := dbutil.NewTransactionHelper(conn)

	t.Run("RunInTx should use write connection", func(t *testing.T) {
		mockConn := conn.(*testutils.MockDBConnection)
		mockConn.ResetCounters()

		// * Run a transaction
		err := txHelper.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
			// * Transaction logic here
			return nil
		})
		require.NoError(t, err)

		// * Verify only write connection was used
		readCount, writeCount := testutils.GetOperationCounts(t, conn)
		assert.Equal(t, int64(0), readCount, "Transaction should not use read connection")
		assert.Greater(t, writeCount, int64(0), "Transaction should use write connection")
	})

	t.Run("Transaction with replica failure", func(t *testing.T) {
		// * Simulate replica failure
		testutils.SimulateReplicaFailure(t, conn, true)
		defer testutils.SimulateReplicaFailure(t, conn, false)

		// * Transaction should still work
		err := txHelper.RunInTx(ctx, func(ctx context.Context, tx bun.Tx) error {
			return nil
		})
		require.NoError(t, err)
	})
}
