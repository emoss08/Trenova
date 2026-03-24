//go:build integration

package validationframework

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestTable(t *testing.T, tc *testutil.TestContext, db interface {
	ExecContext(ctx interface{}, query string, args ...interface{}) (interface{}, error)
}) {
	t.Helper()

	_, err := db.(interface {
		ExecContext(ctx interface{}, query string, args ...interface{}) (interface{}, error)
	}).ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_uniqueness_entities (
			id VARCHAR(100) PRIMARY KEY,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL,
			code VARCHAR(50) NOT NULL
		)`,
	)
	require.NoError(t, err)
}

func TestBunUniquenessChecker_CheckUniqueness_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	_, err := db.ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_uniqueness_entities (
			id VARCHAR(100) PRIMARY KEY,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL,
			code VARCHAR(50) NOT NULL
		)`,
	)
	require.NoError(t, err)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	entityID := pulid.MustNew("ent_")

	_, err = db.ExecContext(
		tc.Ctx,
		`INSERT INTO test_uniqueness_entities (id, organization_id, business_unit_id, name, code)
		 VALUES (?, ?, ?, ?, ?)`,
		entityID.String(), orgID.String(), buID.String(), "Existing Entity", "CODE001",
	)
	require.NoError(t, err)

	checker := NewBunUniquenessChecker(db)

	t.Run("returns false when no conflict exists", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "New Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("returns true when conflict exists", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("case insensitive match finds conflict", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "existing entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("case sensitive match does not find conflict with different case", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "existing entity", CaseSensitive: true},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("case sensitive match finds exact match", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: true},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("excludes current entity on update", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ExcludeID:      entityID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("finds conflict with different entity same name", func(t *testing.T) {
		otherEntityID := pulid.MustNew("ent_")
		_, err := db.ExecContext(
			tc.Ctx,
			`INSERT INTO test_uniqueness_entities (id, organization_id, business_unit_id, name, code)
			 VALUES (?, ?, ?, ?, ?)`,
			otherEntityID.String(),
			orgID.String(),
			buID.String(),
			"Another Entity",
			"CODE002",
		)
		require.NoError(t, err)

		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ExcludeID:      entityID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Another Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("respects organization boundary", func(t *testing.T) {
		otherOrgID := pulid.MustNew("org_")

		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: otherOrgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("respects business unit boundary", func(t *testing.T) {
		otherBuID := pulid.MustNew("bu_")

		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: otherBuID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("checks multiple fields", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
				{Column: "code", Value: "CODE001", CaseSensitive: true},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("multiple fields with one mismatch returns false", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
				{Column: "code", Value: "DIFFERENT", CaseSensitive: true},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("works without organization filter", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			BusinessUnitID: buID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("works without business unit filter", func(t *testing.T) {
		req := &UniquenessRequest{
			TableName:      "test_uniqueness_entities",
			OrganizationID: orgID,
			Fields: []FieldCheck{
				{Column: "name", Value: "Existing Entity", CaseSensitive: false},
			},
		}

		exists, err := checker.CheckUniqueness(tc.Ctx, req)

		require.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestBunUniquenessChecker_CheckUniqueness_EmptyTable_Integration(t *testing.T) {
	testutil.RequireIntegration(t)

	tc, db := testutil.SetupTestDB(t)

	_, err := db.ExecContext(
		tc.Ctx,
		`CREATE TABLE IF NOT EXISTS test_empty_entities (
			id VARCHAR(100) PRIMARY KEY,
			organization_id VARCHAR(100) NOT NULL,
			business_unit_id VARCHAR(100) NOT NULL,
			name VARCHAR(100) NOT NULL
		)`,
	)
	require.NoError(t, err)

	checker := NewBunUniquenessChecker(db)

	req := &UniquenessRequest{
		TableName:      "test_empty_entities",
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Fields: []FieldCheck{
			{Column: "name", Value: "Any Name", CaseSensitive: false},
		},
	}

	exists, err := checker.CheckUniqueness(tc.Ctx, req)

	require.NoError(t, err)
	assert.False(t, exists)
}
