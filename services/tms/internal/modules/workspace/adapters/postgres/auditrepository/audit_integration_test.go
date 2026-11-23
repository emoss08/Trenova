//go:build integration

package auditrepository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fixture struct {
	ctx  context.Context
	db   *keel.DB
	conn *postgres.Connection
	repo auditports.AuditRepository
	data *seedtest.TestData
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	conn := postgres.WrapDB(db)

	return &fixture{
		ctx:  data.Ctx(ctx),
		db:   db,
		conn: conn,
		repo: New(Params{DB: conn, Logger: zap.NewNop()}),
		data: data,
	}
}

func (f *fixture) entry(
	resource permission.Resource,
	resourceID string,
	category audit.Category,
) *audit.Entry {
	e := &audit.Entry{
		ID:             pulid.MustNew("ae_"),
		OrganizationID: f.data.Organization.ID,
		BusinessUnitID: f.data.BusinessUnit.ID,
		Timestamp:      time.Now().Unix(),
		Resource:       resource,
		ResourceID:     resourceID,
		Operation:      permission.OpUpdate,
		Category:       category,
		PrincipalType:  "session_user",
		PrincipalID:    f.data.User.ID,
		UserID:         f.data.User.ID,
		Changes:        map[string]any{"status": map[string]any{"from": "New", "to": "Assigned"}},
	}
	e.Normalize()
	return e
}

func (f *fixture) list(t *testing.T, req *auditports.ListAuditEntriesRequest) []*audit.Entry {
	t.Helper()

	if req.Filter == nil {
		req.Filter = &pagination.QueryOptions{
			TenantInfo: f.data.Info(),
			Pagination: pagination.Info{Limit: 50},
		}
	}
	result, err := f.repo.ListConnection(f.ctx, req)
	require.NoError(t, err)
	return result.Items
}

func TestInsert_NamesTheActorAsTheyWereWhenTheEntryWasWritten(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	got, err := f.repo.GetByID(f.ctx, &auditports.GetAuditEntryByIDOptions{
		EntryID: e.ID,
		Access:  auditports.EntryAccess{AllResources: true},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, got.ActorName, "the insert trigger names the user")
	assert.Equal(t, "shipment.update", got.Action)
	assert.Equal(t, audit.ChannelWeb, got.Channel)
}

func TestInsert_JoinsTheCallersTransactionAndRollsBackWithIt(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_rolled_back", audit.CategoryUser)
	failure := errors.New("the business write failed after the entry was recorded")
	err := dbtx.WriteErr(f.ctx, f.conn, func(ctx context.Context) error {
		require.NoError(t, f.repo.Insert(ctx, []*audit.Entry{e}))
		return failure
	})
	require.ErrorIs(t, err, failure)

	_, err = f.repo.GetByID(f.ctx, &auditports.GetAuditEntryByIDOptions{
		EntryID: e.ID,
		Access:  auditports.EntryAccess{AllResources: true},
	})
	require.Error(t, err, "an entry written in a transaction that rolled back is not kept")
}

func TestListConnection_ShowsOnlyTheResourcesTheReaderCanReadAndHidesSecurity(t *testing.T) {
	f := newFixture(t)

	shipment := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	invoice := f.entry(permission.ResourceInvoice, "inv_1", audit.CategoryUser)
	role := f.entry(permission.ResourceRole, "rol_1", audit.CategorySecurity)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{shipment, invoice, role}))

	items := f.list(t, &auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{
		Resources: []permission.Resource{permission.ResourceShipment, permission.ResourceRole},
	}})
	require.Len(t, items, 1)
	assert.Equal(t, shipment.ID, items[0].ID)

	items = f.list(t, &auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{
		AllResources:    true,
		IncludeSecurity: true,
	}})
	assert.Len(t, items, 3)

	assert.Empty(
		t,
		f.list(t, &auditports.ListAuditEntriesRequest{}),
		"no readable resource reads nothing",
	)
}

func TestListConnection_LeavesTheStatesOutOfASummaryPage(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	summary := f.list(
		t,
		&auditports.ListAuditEntriesRequest{Access: auditports.EntryAccess{AllResources: true}},
	)
	require.Len(t, summary, 1)
	assert.Nil(t, summary[0].Changes)
	assert.Equal(t, "shipment.update", summary[0].Action)

	detail := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:        auditports.EntryAccess{AllResources: true},
		IncludeDetail: true,
	})
	require.Len(t, detail, 1)
	assert.Contains(t, detail[0].Changes, "status")
}

func TestListConnection_FieldHistoryKeepsOnlyEntriesThatChangedTheField(t *testing.T) {
	f := newFixture(t)

	status := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	weight := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	weight.Changes = map[string]any{"weight": map[string]any{"from": 1, "to": 2}}
	other := f.entry(permission.ResourceShipment, "shp_2", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{status, weight, other}))

	items := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:       auditports.EntryAccess{AllResources: true},
		ResourceID:   "shp_1",
		ChangedField: "weight",
	})
	require.Len(t, items, 1)
	assert.Equal(t, weight.ID, items[0].ID)
}

func TestPartitionHorizon_ReachesTwoYearsPastTheMigration(t *testing.T) {
	f := newFixture(t)

	horizon, err := f.repo.PartitionHorizon(f.ctx)
	require.NoError(t, err)
	assert.Greater(t, horizon, time.Now().AddDate(1, 11, 0).Unix(),
		"the migration creates partitions 24 months ahead")

	future := f.entry(permission.ResourceShipment, "shp_future", audit.CategoryUser)
	future.Timestamp = time.Now().AddDate(1, 6, 0).Unix()
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{future}))
}

func TestTenantsWithEntries_NamesTheTenantsWithEntriesInTheWindow(t *testing.T) {
	f := newFixture(t)

	e := f.entry(permission.ResourceShipment, "shp_1", audit.CategoryUser)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{e}))

	tenants, err := f.repo.TenantsWithEntries(f.ctx, e.Timestamp-60, e.Timestamp+60)
	require.NoError(t, err)
	assert.Contains(t, tenants, auditports.TenantKey{
		OrganizationID: f.data.Organization.ID,
		BusinessUnitID: f.data.BusinessUnit.ID,
	})

	tenants, err = f.repo.TenantsWithEntries(f.ctx, e.Timestamp+60, e.Timestamp+120)
	require.NoError(t, err)
	assert.Empty(t, tenants)
}

func (f *fixture) subjectEntry(
	t *testing.T,
	resource permission.Resource,
	resourceID string,
	workerID pulid.ID,
	at int64,
) *audit.Entry {
	e := f.entry(resource, resourceID, audit.CategoryUser)
	e.Timestamp = at
	require.NoError(t, e.SetSubject(permission.ResourceWorker, workerID))
	e.Normalize()
	return e
}

func TestListConnection_BySubjectReadsTheWholeFileNewestFirst(t *testing.T) {
	f := newFixture(t)

	workerID := pulid.MustNew("wrk_")
	otherWorker := pulid.MustNew("wrk_")
	now := time.Now().Unix()

	root := f.subjectEntry(t, permission.ResourceWorker, workerID.String(), workerID, now-30)
	credential := f.subjectEntry(t, permission.ResourceWorkerCredential, "wcr_1", workerID, now-20)
	test := f.subjectEntry(t, permission.ResourceWorkerDOTTest, "wdt_1", workerID, now-10)
	unrelated := f.subjectEntry(t, permission.ResourceWorkerCredential, "wcr_2", otherWorker, now-5)
	unattributed := f.entry(permission.ResourceWorkerCredential, "wcr_3", audit.CategoryUser)
	require.NoError(
		t,
		f.repo.Insert(f.ctx, []*audit.Entry{root, credential, test, unrelated, unattributed}),
	)

	items := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:    auditports.EntryAccess{AllResources: true},
		SubjectID: workerID,
	})
	require.Len(t, items, 3)
	assert.Equal(t, []pulid.ID{test.ID, credential.ID, root.ID}, entryIDs(items))
	for _, item := range items {
		assert.Equal(
			t,
			permission.ResourceWorker,
			item.SubjectResource,
			"the summary carries the subject",
		)
		assert.Equal(t, workerID, item.SubjectID)
	}

	items = f.list(t, &auditports.ListAuditEntriesRequest{
		Access: auditports.EntryAccess{Resources: []permission.Resource{
			permission.ResourceWorker,
			permission.ResourceWorkerCredential,
		}},
		SubjectID: workerID,
	})
	assert.Equal(t, []pulid.ID{credential.ID, root.ID}, entryIDs(items),
		"a reader without worker_dot_test read sees none of the worker's tests")
}

func TestListConnection_BySubjectNeverReadsAnotherTenantsEntries(t *testing.T) {
	f := newFixture(t)
	other := seedtest.SeedAdditionalTenant(t, f.ctx, f.db, "QZ")

	workerID := pulid.MustNew("wrk_")
	mine := f.subjectEntry(
		t,
		permission.ResourceWorker,
		workerID.String(),
		workerID,
		time.Now().Unix(),
	)
	theirs := f.subjectEntry(
		t,
		permission.ResourceWorker,
		workerID.String(),
		workerID,
		time.Now().Unix(),
	)
	theirs.OrganizationID = other.Organization.ID
	theirs.BusinessUnitID = other.BusinessUnit.ID
	theirs.PrincipalID = other.User.ID
	theirs.UserID = other.User.ID
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{mine}))
	require.NoError(t, f.repo.Insert(other.Ctx(f.ctx), []*audit.Entry{theirs}))

	items := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:    auditports.EntryAccess{AllResources: true, IncludeSecurity: true},
		SubjectID: workerID,
	})
	assert.Equal(t, []pulid.ID{mine.ID}, entryIDs(items))
}

func TestListConnection_BySubjectPagesWithoutRepeatingOrSkipping(t *testing.T) {
	f := newFixture(t)

	workerID := pulid.MustNew("wrk_")
	at := time.Now().Unix()
	entries := make([]*audit.Entry, 0, 5)
	for range 5 {
		entries = append(
			entries,
			f.subjectEntry(
				t,
				permission.ResourceWorkerCredential,
				pulid.MustNew("wcr_").String(),
				workerID,
				at,
			),
		)
	}
	require.NoError(t, f.repo.Insert(f.ctx, entries))

	seen := make(map[pulid.ID]struct{}, len(entries))
	after := ""
	for page := 0; ; page++ {
		require.Less(t, page, 5, "the pages end")
		result, err := f.repo.ListConnection(f.ctx, &auditports.ListAuditEntriesRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: f.data.Info(),
				Pagination: pagination.Info{Limit: 2},
			},
			Cursor:    pagination.CursorInfo{Limit: 2, After: after},
			Access:    auditports.EntryAccess{AllResources: true},
			SubjectID: workerID,
		})
		require.NoError(t, err)
		for _, item := range result.Items {
			_, repeated := seen[item.ID]
			require.False(t, repeated, "entry %s appears on two pages", item.ID)
			seen[item.ID] = struct{}{}
		}
		if !result.HasNextPage {
			break
		}
		last, ok := result.EdgeCursors.EdgeCursorAt(len(result.Items) - 1)
		require.True(t, ok)
		after = last
	}
	assert.Len(
		t,
		seen,
		len(entries),
		"entries sharing a timestamp are ordered by id, so none is skipped",
	)
}

func TestListConnection_BySubjectReadsTheSubjectIndex(t *testing.T) {
	f := newFixture(t)

	e := auditdb.Entries
	query := e.Where(recordPreds(&auditports.ListAuditEntriesRequest{SubjectID: pulid.MustNew("wrk_")})...).
		Where(e.Category.Ne(audit.CategorySecurity)).
		Order(e.Timestamp.Desc(), e.ID.Desc()).
		Limit(51)

	var plan *keel.Plan
	err := keel.Read(f.ctx, f.db, func(ctx context.Context) error {
		var explainErr error
		plan, explainErr = query.Explain(ctx, f.db, keel.ExplainOptions{GenericPlan: true})
		return explainErr
	}, keel.Setting("enable_seqscan", "off"))
	require.NoError(t, err)

	partitionIndexes := subjectIndexPartitions(t, f)
	require.NotEmpty(t, partitionIndexes, "every partition carries the subject index")
	require.NotEmpty(t, plan.Indexes(), "the read uses an index\n%s", plan.JSON)
	for _, index := range plan.Indexes() {
		assert.Contains(t, partitionIndexes, index,
			"the read uses only the subject index's partitions\n%s", plan.JSON)
	}
}

func TestEnsurePartitions_GivesANewPartitionTheSubjectColumnsAndIndex(t *testing.T) {
	f := newFixture(t)
	sys := dbscope.WithSystem(f.ctx, "create and inspect an audit partition in a test")

	var created, earliest int64
	err := keel.Write(sys, f.db, func(ctx context.Context) error {
		bounds, err := keel.QueryAll(ctx, f.db, pgx.RowTo[int64], `
			SELECT min((regexp_match(pg_get_expr(c.relpartbound, c.oid), 'FROM \(''?(-?\d+)''?\)'))[1]::BIGINT)
			FROM pg_inherits i
			JOIN pg_class c ON c.oid = i.inhrelid
			WHERE i.inhparent = 'public.audit_entries'::regclass`)
		if err != nil {
			return err
		}
		require.Len(t, bounds, 1)
		earliest = bounds[0]
		rows, err := keel.QueryAll(ctx, f.db, pgx.RowTo[int64],
			`SELECT audit_partitions.ensure_partitions($1, 1)::BIGINT`, earliest-15*24*60*60)
		if err != nil {
			return err
		}
		created = rows[0]
		return nil
	})
	require.NoError(t, err)
	require.Positive(t, created, "a month before the earliest partition is created")

	missing, err := keel.QueryAll(sys, f.db, pgx.RowTo[string], `
		SELECT c.relname::TEXT
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'public.audit_entries'::regclass
			AND (
				(SELECT count(*) FROM pg_attribute a
				 WHERE a.attrelid = c.oid AND NOT a.attisdropped
					AND a.attname IN ('subject_resource', 'subject_id')) <> 2
				OR NOT EXISTS (
					SELECT 1 FROM pg_index x
					JOIN pg_inherits xi ON xi.inhrelid = x.indexrelid
					WHERE x.indrelid = c.oid
						AND xi.inhparent = 'public.idx_audit_entries_tenant_subject'::regclass
				)
			)`)
	require.NoError(t, err)
	assert.Empty(
		t,
		missing,
		"every partition, the new one included, has the subject columns and index",
	)

	workerID := pulid.MustNew("wrk_")
	old := f.subjectEntry(t, permission.ResourceWorker, workerID.String(), workerID,
		earliest-15*24*60*60)
	require.NoError(t, f.repo.Insert(f.ctx, []*audit.Entry{old}))
	items := f.list(t, &auditports.ListAuditEntriesRequest{
		Access:    auditports.EntryAccess{AllResources: true},
		SubjectID: workerID,
	})
	assert.Equal(t, []pulid.ID{old.ID}, entryIDs(items))
}

func subjectIndexPartitions(t *testing.T, f *fixture) []string {
	t.Helper()

	sys := dbscope.WithSystem(f.ctx, "list the subject index's partitions in a test")
	names, err := keel.QueryAll(sys, f.db, pgx.RowTo[string], `
		SELECT c.relname::TEXT
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		WHERE i.inhparent = 'public.idx_audit_entries_tenant_subject'::regclass`)
	require.NoError(t, err)
	return names
}

func entryIDs(entries []*audit.Entry) []pulid.ID {
	out := make([]pulid.ID, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.ID)
	}
	return out
}
