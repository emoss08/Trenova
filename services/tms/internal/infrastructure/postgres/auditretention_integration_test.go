//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func daysAgo(days int) int64 {
	return time.Now().AddDate(0, 0, -days).Unix()
}

func (f *rlsFixture) seedAuditEntry(t *testing.T, critical bool, timestamp int64) string {
	t.Helper()

	id := pulid.MustNew("ae_").String()
	_, err := f.admin.NewRaw(`
		INSERT INTO audit_entries (id, business_unit_id, organization_id, resource, resource_id, operation,
			user_id, principal_type, principal_id, timestamp, critical)
		VALUES (?, ?, ?, 'user', 'res', 'update', ?, 'session_user', ?, ?, ?)`,
		id, f.tenantA.BusinessUnitID, f.tenantA.OrganizationID, f.tenantA.UserID, f.tenantA.UserID,
		timestamp, critical,
	).Exec(t.Context())
	require.NoError(t, err)

	return id
}

func (f *rlsFixture) auditEntryIDs(t *testing.T) []string {
	t.Helper()

	var ids []string
	err := f.admin.NewRaw("SELECT id FROM audit_entries WHERE organization_id = ? ORDER BY id",
		f.tenantA.OrganizationID).Scan(t.Context(), &ids)
	require.NoError(t, err)

	return ids
}

func deleteAuditEntriesBefore(before, criticalBefore int64) func(bun.Tx) *bun.DeleteQuery {
	return func(tx bun.Tx) *bun.DeleteQuery {
		return tx.NewDelete().TableExpr("audit_entries").
			Where("timestamp < ?", before).
			WhereGroup(" AND ", func(q *bun.DeleteQuery) *bun.DeleteQuery {
				return q.Where("NOT critical").WhereOr("timestamp < ?", criticalBefore)
			})
	}
}

func TestAuditRetention_TheApplicationRoleCannotDeleteOrUpdate(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, false, daysAgo(400))

	ctx := dbscope.WithTenant(t.Context(), f.tenantA)
	for _, statement := range []string{
		"DELETE FROM audit_entries",
		"UPDATE audit_entries SET comment = 'rewritten'",
		"DELETE FROM auth_events",
	} {
		err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
			_, err := tx.ExecContext(ctx, statement)
			return err
		})
		require.Error(t, err, statement)
		assert.Contains(t, err.Error(), "permission denied", statement)
	}

	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_TheSystemRoleMustOptInToDelete(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, false, daysAgo(400))

	ctx := dbscope.WithSystem(t.Context(), "integration test retention")
	err := conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM audit_entries")
		return err
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "append-only")

	err = conn.WithTx(ctx, ports.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, enableAuditRetentionSQL); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE audit_entries SET comment = 'rewritten'")
		return err
	})
	require.Error(t, err, "retention never permits an update")

	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_KeepsCriticalEntriesForAYear(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)

	oldRoutine := f.seedAuditEntry(t, false, daysAgo(400))
	recentRoutine := f.seedAuditEntry(t, false, daysAgo(10))
	oldCritical := f.seedAuditEntry(t, true, daysAgo(400))
	recentCritical := f.seedAuditEntry(t, true, daysAgo(100))

	ctx := dbscope.WithSystem(t.Context(), "integration test retention")

	_, err := DeleteUnderAuditRetention(ctx, conn, deleteAuditEntriesBefore(daysAgo(30), daysAgo(30)))
	require.Error(t, err, "a critical entry younger than a year is refused even when asked for")
	assert.Len(t, f.auditEntryIDs(t), 4, "the refused sweep rolls back as a whole")

	deleted, err := DeleteUnderAuditRetention(ctx, conn, deleteAuditEntriesBefore(daysAgo(30), daysAgo(365)))
	require.NoError(t, err)
	assert.EqualValues(t, 2, deleted)

	remaining := f.auditEntryIDs(t)
	assert.ElementsMatch(t, []string{recentRoutine, recentCritical}, remaining)
	assert.NotContains(t, remaining, oldRoutine)
	assert.NotContains(t, remaining, oldCritical)
}

func TestAuditRetention_AuthEventsAreAppendOnly(t *testing.T) {
	f := newRLSFixture(t, config.RLSModeEnforce)
	conn := f.connect(t)
	ctx := t.Context()

	insert := func(occurredAt int64) string {
		id := pulid.MustNew("aue_").String()
		_, err := f.admin.NewRaw(`
			INSERT INTO auth_events (id, user_id, organization_id, business_unit_id, provider, outcome, occurred_at)
			VALUES (?, ?, ?, ?, 'password', 'success', ?)`,
			id, f.tenantA.UserID, f.tenantA.OrganizationID, f.tenantA.BusinessUnitID, occurredAt,
		).Exec(ctx)
		require.NoError(t, err)
		return id
	}
	recent := insert(daysAgo(5))
	expired := insert(daysAgo(400))

	_, err := f.admin.NewRaw("UPDATE auth_events SET outcome = 'failed' WHERE id = ?", recent).Exec(ctx)
	require.Error(t, err)

	_, err = f.admin.NewRaw("UPDATE auth_events SET user_id = NULL WHERE id = ?", recent).Exec(ctx)
	require.NoError(t, err, "a deleted user or identity provider is unlinked by its foreign key")

	_, err = f.admin.NewRaw("DELETE FROM auth_events WHERE id = ?", expired).Exec(ctx)
	require.Error(t, err)

	systemCtx := dbscope.WithSystem(ctx, "integration test retention")
	deleted, err := DeleteUnderAuditRetention(systemCtx, conn, func(tx bun.Tx) *bun.DeleteQuery {
		return tx.NewDelete().TableExpr("auth_events").Where("occurred_at < ?", daysAgo(365))
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)

	_, err = DeleteUnderAuditRetention(systemCtx, conn, func(tx bun.Tx) *bun.DeleteQuery {
		return tx.NewDelete().TableExpr("auth_events").Where("id = ?", recent)
	})
	require.Error(t, err, "an event younger than the retention period cannot be deleted")
}
