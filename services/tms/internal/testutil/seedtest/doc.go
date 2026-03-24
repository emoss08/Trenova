// Package seedtest provides testing utilities for database seeding tests.
//
// This package follows DRY (Don't Repeat Yourself) principles by providing
// reusable components for seed testing:
//
// # Test Database Setup
//
// TestDB provides transaction-based test isolation:
//
//	func TestMySeed(t *testing.T) {
//		db := // get your test database
//		tdb := seedtest.NewTestDB(t, db)
//		defer tdb.Rollback()
//
//		// Run tests with tdb.Tx
//	}
//
// # Entity Builders
//
// Fluent builders for creating test entities:
//
//	state := seedtest.NewState().WithAbbreviation("TX").Build(t, ctx, tx)
//	bu := seedtest.NewBusinessUnit().WithCode("TEST").Build(t, ctx, tx)
//	org := seedtest.NewOrganization(bu.ID, state.ID).
//		WithScacCode("TEST").
//		Build(t, ctx, tx)
//
// # Assertions
//
// Helper functions for common assertions:
//
//	seedtest.AssertOrganizationExists(t, ctx, db, orgID)
//	seedtest.AssertUserHasFields(t, user, "John Doe", "jdoe", "john@example.com")
//	seedtest.AssertEntityCount(t, ctx, db, "organizations", 1)
//
// # Mock Seed Context
//
// Enhanced SeedContext for testing:
//
//	mockCtx := seedtest.NewMockSeedContext(t, db)
//	mockCtx.RequireSet("test_key", value)
//	mockCtx.AssertKeyExists("test_key")
//	mockCtx.AssertTrackedEntityCount("MySeed", 5)
//
// # Complete Test Data
//
// Create all required entities in one call:
//
//	data := seedtest.CreateFullTestData(t, ctx, tx)
//	// data.State, data.BusinessUnit, data.Organization, data.User
package seedtest
