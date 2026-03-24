package seedtest

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func AssertBusinessUnitExists(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	id pulid.ID,
) *tenant.BusinessUnit {
	t.Helper()

	var bu tenant.BusinessUnit
	err := db.NewSelect().
		Model(&bu).
		Where("id = ?", id).
		Scan(ctx)

	require.NoError(t, err, "business unit should exist in database")
	return &bu
}

func AssertOrganizationExists(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	id pulid.ID,
) *tenant.Organization {
	t.Helper()

	var org tenant.Organization
	err := db.NewSelect().
		Model(&org).
		Where("id = ?", id).
		Scan(ctx)

	require.NoError(t, err, "organization should exist in database")
	return &org
}

func AssertUserExists(t *testing.T, ctx context.Context, db bun.IDB, id pulid.ID) *tenant.User {
	t.Helper()

	var user tenant.User
	err := db.NewSelect().
		Model(&user).
		Where("id = ?", id).
		Scan(ctx)

	require.NoError(t, err, "user should exist in database")
	return &user
}

func AssertStateExists(
	t *testing.T,
	ctx context.Context,
	db bun.IDB,
	abbreviation string,
) *usstate.UsState {
	t.Helper()

	var state usstate.UsState
	err := db.NewSelect().
		Model(&state).
		Where("abbreviation = ?", abbreviation).
		Scan(ctx)

	require.NoError(t, err, "state should exist in database")
	return &state
}

func AssertBusinessUnitNotExists(t *testing.T, ctx context.Context, db bun.IDB, id pulid.ID) {
	t.Helper()

	var bu tenant.BusinessUnit
	err := db.NewSelect().
		Model(&bu).
		Where("id = ?", id).
		Scan(ctx)

	assert.Error(t, err, "business unit should not exist in database")
}

func AssertOrganizationNotExists(t *testing.T, ctx context.Context, db bun.IDB, id pulid.ID) {
	t.Helper()

	var org tenant.Organization
	err := db.NewSelect().
		Model(&org).
		Where("id = ?", id).
		Scan(ctx)

	assert.Error(t, err, "organization should not exist in database")
}

func AssertUserNotExists(t *testing.T, ctx context.Context, db bun.IDB, id pulid.ID) {
	t.Helper()

	var user tenant.User
	err := db.NewSelect().
		Model(&user).
		Where("id = ?", id).
		Scan(ctx)

	assert.Error(t, err, "user should not exist in database")
}

func AssertEntityCount(t *testing.T, ctx context.Context, db bun.IDB, table string, expected int) {
	t.Helper()

	count, err := db.NewSelect().
		Table(table).
		Count(ctx)

	require.NoError(t, err, "failed to count entities in table %s", table)
	assert.Equal(t, expected, count, "entity count in table %s should match", table)
}

func AssertOrganizationHasFields(
	t *testing.T,
	org *tenant.Organization,
	name, scacCode, dotNumber string,
) {
	t.Helper()

	assert.Equal(t, name, org.Name, "organization name should match")
	assert.Equal(t, scacCode, org.ScacCode, "organization SCAC code should match")
	assert.Equal(t, dotNumber, org.DOTNumber, "organization DOT number should match")
}

func AssertUserHasFields(t *testing.T, user *tenant.User, name, username, email string) {
	t.Helper()

	assert.Equal(t, name, user.Name, "user name should match")
	assert.Equal(t, username, user.Username, "user username should match")
	assert.Equal(t, email, user.EmailAddress, "user email should match")
}

func AssertUserPasswordMatches(t *testing.T, expectedHash, actualHash string) {
	t.Helper()

	assert.Equal(t, expectedHash, actualHash, "user password hash should match")
}
