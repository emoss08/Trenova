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

type seededAuditEntry struct {
	class     string
	critical  bool
	resource  string
	recordID  string
	timestamp int64
}

func (f *rlsFixture) seedAuditEntry(t *testing.T, e seededAuditEntry) string {
	t.Helper()

	if e.resource == "" {
		e.resource = "user"
	}
	if e.recordID == "" {
		e.recordID = "res"
	}
	_, err := keel.Exec(t.Context(), f.admin,
		"SELECT audit_partitions.ensure_partitions($1, 3)", e.timestamp)
	require.NoError(t, err)

	id := pulid.MustNew("ae_").String()
	_, err = keel.Exec(t.Context(), f.admin, `
		INSERT INTO audit_entries (id, business_unit_id, organization_id, resource, resource_id, operation,
			action, category, retention_class, channel, user_id, principal_type, principal_id, timestamp, critical)
		VALUES ($1, $2, $3, $4::varchar, $5, 'update', $4::varchar || '.update', 'User', $6, 'web', $7, 'session_user', $7, $8, $9)`,
		id, f.tenantA.BusinessUnitID.String(), f.tenantA.OrganizationID.String(),
		e.resource, e.recordID, e.class, f.tenantA.UserID.String(), e.timestamp, e.critical,
	)
	require.NoError(t, err)

	return id
}

func (f *rlsFixture) placeLegalHold(t *testing.T, resource, recordID string) string {
	t.Helper()

	id := pulid.MustNew("alh_").String()
	_, err := keel.Exec(t.Context(), f.admin, `
		INSERT INTO audit_legal_holds (id, business_unit_id, organization_id, name, reason, resource,
			resource_id, placed_by_id, placed_at)
		VALUES ($1, $2, $3, 'Litigation', 'Preserve for counsel', NULLIF($4, ''), NULLIF($5, ''), $6, $7)`,
		id, f.tenantA.BusinessUnitID.String(), f.tenantA.OrganizationID.String(),
		resource, recordID, f.tenantA.UserID.String(), time.Now().Unix(),
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

func deleteAuditEntries(where string, args ...any) AuditRetentionDelete {
	return func(ctx context.Context, db *keel.DB) (int64, error) {
		return keel.Exec(ctx, db, "DELETE FROM audit_entries WHERE "+where, args...)
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

func TestAuditRetention_TheApplicationRoleCannotDeleteUpdateOrReachPartitions(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, seededAuditEntry{class: "Access", timestamp: daysAgo(400)})

	partition, err := queryStrings(t.Context(), f.admin, `
		SELECT format('audit_partitions.%I', c.relname)
		FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'public.audit_entries'::regclass
		ORDER BY c.relname LIMIT 1`)
	require.NoError(t, err)
	require.Len(t, partition, 1)

	ctx := dbscope.WithTenant(t.Context(), f.tenantA)
	for _, statement := range []string{
		"DELETE FROM audit_entries",
		"UPDATE audit_entries SET comment = 'rewritten'",
		"DELETE FROM auth_events",
		"SELECT count(*) FROM " + partition[0],
		"DELETE FROM " + partition[0],
		"SELECT audit_partitions.ensure_partitions(NULL, 3)",
	} {
		err := execInTransaction(ctx, conn, statement)
		require.Error(t, err, statement)
		assert.Contains(t, err.Error(), "permission denied", statement)
	}

	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_TheTableCannotBeTruncated(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	f.seedAuditEntry(t, seededAuditEntry{class: "Access", timestamp: daysAgo(400)})

	_, err := keel.Exec(t.Context(), f.admin, "TRUNCATE audit_entries")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be truncated")
	assert.Len(t, f.auditEntryIDs(t), 1)
}

func TestAuditRetention_TheSystemRoleMustOptInToDelete(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	f.seedAuditEntry(t, seededAuditEntry{class: "Access", timestamp: daysAgo(400)})

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

func TestAuditRetention_KeepsEachClassForItsFloor(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithSystem(t.Context(), "integration test retention")

	expiredAccess := f.seedAuditEntry(t, seededAuditEntry{class: "Access", timestamp: daysAgo(100)})
	recentAccess := f.seedAuditEntry(t, seededAuditEntry{class: "Access", timestamp: daysAgo(30)})
	expiredSecurity := f.seedAuditEntry(
		t,
		seededAuditEntry{class: "Security", critical: true, timestamp: daysAgo(800)},
	)
	youngSecurity := f.seedAuditEntry(
		t,
		seededAuditEntry{class: "Security", critical: true, timestamp: daysAgo(400)},
	)
	youngFinancial := f.seedAuditEntry(t, seededAuditEntry{class: "Financial", timestamp: daysAgo(2000)})

	for _, id := range []string{recentAccess, youngSecurity, youngFinancial} {
		_, err := DeleteUnderAuditRetention(ctx, conn, deleteAuditEntries("id = $1", id))
		require.Error(t, err, "an entry younger than its class's floor is refused even when asked for")
	}
	assert.Len(t, f.auditEntryIDs(t), 5)

	deleted, err := DeleteUnderAuditRetention(
		ctx,
		conn,
		deleteAuditEntries("id IN ($1, $2)", expiredAccess, expiredSecurity),
	)
	require.NoError(t, err)
	assert.EqualValues(t, 2, deleted)
	assert.ElementsMatch(t, []string{recentAccess, youngSecurity, youngFinancial}, f.auditEntryIDs(t))
}

func TestAuditRetention_ALegalHoldStopsDeletionUntilReleased(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)
	ctx := dbscope.WithSystem(t.Context(), "integration test retention")

	held := f.seedAuditEntry(t, seededAuditEntry{
		class: "Access", resource: "shipment", recordID: "shp_1", timestamp: daysAgo(200),
	})
	other := f.seedAuditEntry(t, seededAuditEntry{
		class: "Access", resource: "shipment", recordID: "shp_2", timestamp: daysAgo(200),
	})
	hold := f.placeLegalHold(t, "shipment", "shp_1")

	_, err := DeleteUnderAuditRetention(ctx, conn, deleteAuditEntries("id = $1", held))
	require.Error(t, err, "a held record's entries cannot be deleted")

	deleted, err := DeleteUnderAuditRetention(ctx, conn, deleteAuditEntries("id = $1", other))
	require.NoError(t, err, "a hold on one record leaves the rest to retention")
	assert.EqualValues(t, 1, deleted)

	_, err = keel.Exec(t.Context(), f.admin, "DELETE FROM audit_legal_holds WHERE id = $1", hold)
	require.Error(t, err, "a hold is released, never deleted")

	_, err = keel.Exec(t.Context(), f.admin, `
		UPDATE audit_legal_holds SET released_at = $2, released_by_id = placed_by_id,
			release_reason = 'Matter closed' WHERE id = $1`, hold, time.Now().Unix())
	require.NoError(t, err)

	_, err = keel.Exec(t.Context(), f.admin,
		"UPDATE audit_legal_holds SET name = 'Renamed' WHERE id = $1", hold)
	require.Error(t, err, "a released hold cannot be changed")

	deleted, err = DeleteUnderAuditRetention(ctx, conn, deleteAuditEntries("id = $1", held))
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
}

func TestAuditRetention_PartitionsAreDroppedOnlyPastTenYearsAndWithoutHolds(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	ctx := t.Context()

	_, err := keel.Exec(ctx, f.admin,
		"SELECT audit_partitions.drop_expired_partitions($1)", daysAgo(30))
	require.Error(t, err, "nothing younger than ten years is dropped")

	ancient := time.Now().AddDate(-11, 0, 0).Unix()
	_, err = keel.Exec(ctx, f.admin, "SELECT audit_partitions.ensure_partitions($1, 3)", ancient)
	require.NoError(t, err)
	f.seedAuditEntry(t, seededAuditEntry{class: "Financial", timestamp: ancient})
	hold := f.placeLegalHold(t, "", "")

	dropped, err := queryStrings(ctx, f.admin,
		"SELECT audit_partitions.drop_expired_partitions($1)::text", daysAgo(3651))
	require.NoError(t, err)
	before := len(f.auditEntryIDs(t))
	assert.Equal(t, 1, before, "the held partition stays")
	assert.NotEqual(t, []string{"0"}, dropped, "the empty partitions before it go")

	_, err = keel.Exec(ctx, f.admin, `
		UPDATE audit_legal_holds SET released_at = $2, released_by_id = placed_by_id,
			release_reason = 'Matter closed' WHERE id = $1`, hold, time.Now().Unix())
	require.NoError(t, err)

	_, err = keel.Exec(ctx, f.admin,
		"SELECT audit_partitions.drop_expired_partitions($1)", daysAgo(3651))
	require.NoError(t, err)
	assert.Empty(t, f.auditEntryIDs(t))
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
