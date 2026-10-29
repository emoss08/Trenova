//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func daysAgo(days int) int64 {
	return time.Now().AddDate(0, 0, -days).Unix()
}

func (f *rlsFixture) seedAuditEntry(t *testing.T, critical bool, timestamp int64) string {
	t.Helper()

	id := pulid.MustNew("ae_").String()
	_, err := keel.Exec(t.Context(), f.admin, `
		INSERT INTO audit_entries (id, business_unit_id, organization_id, resource, resource_id, operation,
			user_id, principal_type, principal_id, timestamp, critical)
		VALUES ($1, $2, $3, 'user', 'res', 'update', $4, 'session_user', $4, $5, $6)`,
		id, f.tenantA.BusinessUnitID.String(), f.tenantA.OrganizationID.String(),
		f.tenantA.UserID.String(), timestamp, critical,
	)
	require.NoError(t, err)

	return id
}

func (f *rlsFixture) auditEntryIDs(t *testing.T) []string {
	t.Helper()

	ids, err := queryStrings(t.Context(), f.admin,
		"SELECT id FROM audit_entries WHERE organization_id = $1 ORDER BY id",
		f.tenantA.OrganizationID.String())
	require.NoError(t, err)

	return ids
}

func deleteAuditEntriesBefore(before, criticalBefore int64) AuditRetentionDelete {
	return func(ctx context.Context, db *keel.DB) (int64, error) {
		return keel.Exec(ctx, db,
			"DELETE FROM audit_entries WHERE timestamp < $1 AND (NOT critical OR timestamp < $2)",
			before, criticalBefore)
	}
}

func execInTransaction(ctx context.Context, conn *Connection, statements ...string) error {
	return conn.Transact(ctx, kernelports.TxOptions{}, func(ctx context.Context) error {
		for _, statement := range statements {
			if _, err := keel.Exec(ctx, conn.Keel(ctx), statement); err != nil {
				return err
			}
		}
		return nil
	})
}

func TestAuditRetention_TheApplicationRoleCannotDeleteOrUpdate(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, false, daysAgo(400))

	ctx := dbscope.WithTenant(t.Context(), f.tenantA)
	for _, statement := range []string{
		"DELETE FROM audit_entries",
		"UPDATE audit_entries SET comment = 'rewritten'",
		"DELETE FROM auth_events",
	} {
		err := execInTransaction(ctx, conn, statement)
		require.Error(t, err, statement)
		assert.Contains(t, err.Error(), "permission denied", statement)
	}

	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_TheSystemRoleMustOptInToDelete(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, false, daysAgo(400))

	ctx := dbscope.WithSystem(t.Context(), "integration test retention")
	err := execInTransaction(ctx, conn, "DELETE FROM audit_entries")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	err = execInTransaction(
		ctx,
		conn,
		"SET LOCAL trenova.audit_retention = 'on'",
		"UPDATE audit_entries SET comment = 'rewritten'",
	)
	require.Error(t, err, "retention never permits an update")

	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_KeepsCriticalEntriesForAYear(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	oldRoutine := f.seedAuditEntry(t, false, daysAgo(400))
	recentRoutine := f.seedAuditEntry(t, false, daysAgo(10))
	oldCritical := f.seedAuditEntry(t, true, daysAgo(400))
	recentCritical := f.seedAuditEntry(t, true, daysAgo(100))

	ctx := dbscope.WithSystem(t.Context(), "integration test retention")

	_, err := DeleteUnderAuditRetention(
		ctx,
		conn,
		deleteAuditEntriesBefore(daysAgo(30), daysAgo(30)),
	)
	require.Error(t, err, "a critical entry younger than a year is refused even when asked for")
	assert.Len(t, f.auditEntryIDs(t), 4, "the refused sweep rolls back as a whole")

	deleted, err := DeleteUnderAuditRetention(
		ctx,
		conn,
		deleteAuditEntriesBefore(daysAgo(30), daysAgo(365)),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, deleted)

	remaining := f.auditEntryIDs(t)
	assert.ElementsMatch(t, []string{recentRoutine, recentCritical}, remaining)
	assert.NotContains(t, remaining, oldRoutine)
	assert.NotContains(t, remaining, oldCritical)
}

func TestAuditRetention_AuthEventsAreAppendOnly(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := t.Context()

	insert := func(occurredAt int64) string {
		id := pulid.MustNew("aue_").String()
		_, err := keel.Exec(ctx, f.admin, `
			INSERT INTO auth_events (id, user_id, organization_id, business_unit_id, provider, outcome, occurred_at)
			VALUES ($1, $2, $3, $4, 'password', 'success', $5)`,
			id, f.tenantA.UserID.String(), f.tenantA.OrganizationID.String(),
			f.tenantA.BusinessUnitID.String(), occurredAt,
		)
		require.NoError(t, err)
		return id
	}
	recent := insert(daysAgo(5))
	expired := insert(daysAgo(400))

	_, err := keel.Exec(
		ctx,
		f.admin,
		"UPDATE auth_events SET outcome = 'failed' WHERE id = $1",
		recent,
	)
	require.Error(t, err)

	_, err = keel.Exec(ctx, f.admin, "UPDATE auth_events SET user_id = NULL WHERE id = $1", recent)
	require.NoError(t, err, "a deleted user or identity provider is unlinked by its foreign key")

	_, err = keel.Exec(ctx, f.admin, "DELETE FROM auth_events WHERE id = $1", expired)
	require.Error(t, err)

	systemCtx := dbscope.WithSystem(ctx, "integration test retention")
	deleted, err := DeleteUnderAuditRetention(
		systemCtx,
		conn,
		func(ctx context.Context, db *keel.DB) (int64, error) {
			return keel.Exec(
				ctx,
				db,
				"DELETE FROM auth_events WHERE occurred_at < $1",
				daysAgo(365),
			)
		},
	)
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	_, err = DeleteUnderAuditRetention(
		systemCtx,
		conn,
		func(ctx context.Context, db *keel.DB) (int64, error) {
			return keel.Exec(ctx, db, "DELETE FROM auth_events WHERE id = $1", recent)
		},
	)
	require.Error(t, err, "an event younger than the retention period cannot be deleted")
}
