//go:build integration

package validationframework

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBunReferenceChecker_CheckReference_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	_, err := db.ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_parent_entities (
			id VARCHAR(100) PRIMARY KEY,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL
		)`,
	)
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	parentID := pulid.MustNew("par_")

	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO test_parent_entities (id, organization_id, business_unit_id, name)
		 VALUES (?, ?, ?, ?)`,
		parentID.String(), orgID.String(), buID.String(), "Parent Entity",
	)
	require.NoError(t, err)

	checker := NewBunReferenceChecker(db)

	t.Run("returns true when reference exists", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ID:             parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false when reference does not exist", func(t *testing.T) {
		nonExistentID := pulid.MustNew("par_")
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ID:             nonExistentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("respects organization boundary", func(t *testing.T) {
		otherOrgID := pulid.MustNew("org_")
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			OrganizationID: otherOrgID,
			BusinessUnitID: buID,
			ID:             parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("respects business unit boundary", func(t *testing.T) {
		otherBuID := pulid.MustNew("bu_")
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: otherBuID,
			ID:             parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("works without organization filter", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			BusinessUnitID: buID,
			ID:             parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("works without business unit filter", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_parent_entities",
			OrganizationID: orgID,
			ID:             parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("works without tenant filters", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName: "test_parent_entities",
			ID:        parentID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns error when table name is empty", func(t *testing.T) {
		req := &ReferenceRequest{
			ID: parentID,
		}

		_, err := checker.CheckReference(tc.Ctx, req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "table name is required")
	})

	t.Run("returns error when reference ID is nil", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName: "test_parent_entities",
		}

		_, err := checker.CheckReference(tc.Ctx, req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reference ID is required")
	})
}

func TestBunReferenceChecker_CheckReference_MultipleRecords_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	_, err := db.ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_multi_parent_entities (
			id VARCHAR(100) PRIMARY KEY,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL
		)`,
	)
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	parent1ID := pulid.MustNew("par_")
	parent2ID := pulid.MustNew("par_")
	parent3ID := pulid.MustNew("par_")

	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO test_multi_parent_entities (id, organization_id, business_unit_id, name)
		 VALUES (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?)`,
		parent1ID.String(), orgID.String(), buID.String(), "Parent 1",
		parent2ID.String(), orgID.String(), buID.String(), "Parent 2",
		parent3ID.String(), orgID.String(), buID.String(), "Parent 3",
	)
	require.NoError(t, err)

	checker := NewBunReferenceChecker(db)

	t.Run("finds first parent", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_multi_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ID:             parent1ID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("finds second parent", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_multi_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ID:             parent2ID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("finds third parent", func(t *testing.T) {
		req := &ReferenceRequest{
			TableName:      "test_multi_parent_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ID:             parent3ID,
		}

		exists, err := checker.CheckReference(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})
}
