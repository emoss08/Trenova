//go:build integration

package accountingsyncrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/invoicerepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func driftMinor(v int64) *int64 { return &v }

func (f *syncFixture) driftFinding(
	kind accountingsync.DriftKind,
	objectID pulid.ID,
	at int64,
) *accountingsync.AccountingDriftFinding {
	return accountingsync.NewAccountingDriftFinding(&accountingsync.DriftObservation{
		TenantInfo:    f.tenant,
		ConnectionID:  f.connection.ID,
		ObjectType:    accountingsync.SyncObjectInvoice,
		ObjectID:      objectID,
		ObjectNumber:  "INV-" + objectID.String()[len(objectID.String())-4:],
		PartyName:     "Acme Foods",
		Kind:          kind,
		CurrencyCode:  "USD",
		TrenovaMinor:  driftMinor(10_000),
		ProviderMinor: driftMinor(9_000),
		At:            at,
	})
}

func TestDriftFindingRepository_OneOpenFindingPerDocumentAndKind(t *testing.T) {
	f := setupSyncFixture(t)
	repo := NewDriftFindingRepository(DriftFindingParams{DB: f.conn, Logger: zap.NewNop()})
	invoiceID := pulid.MustNew("inv_")

	first, err := repo.Create(f.ctx, f.driftFinding(accountingsync.DriftAmountMismatch, invoiceID, f.now))
	require.NoError(t, err)
	_, err = repo.Create(f.ctx, f.driftFinding(accountingsync.DriftAmountMismatch, invoiceID, f.now))
	require.Error(t, err, "a second open finding for the same document and kind is refused")

	_, err = repo.Create(f.ctx, f.driftFinding(accountingsync.DriftVoidedInProvider, invoiceID, f.now))
	require.NoError(t, err, "another kind on the same document is its own finding")

	require.True(t, first.Clear(f.now+10))
	_, err = repo.Update(f.ctx, first)
	require.NoError(t, err)
	reopened, err := repo.Create(f.ctx, f.driftFinding(accountingsync.DriftAmountMismatch, invoiceID, f.now+20))
	require.NoError(t, err, "once resolved, the document may drift again")

	stale := *first
	stale.Version = 0
	_, err = repo.Update(f.ctx, &stale)
	require.Error(t, err, "a stale version is refused")

	open, err := repo.ListOpen(f.ctx, &repositories.ListOpenAccountingDriftFindingsRequest{
		TenantInfo:   f.tenant,
		ConnectionID: f.connection.ID,
		ObjectIDs:    []pulid.ID{invoiceID},
	})
	require.NoError(t, err)
	ids := make([]pulid.ID, 0, len(open))
	for _, finding := range open {
		ids = append(ids, finding.ID)
	}
	assert.Len(t, open, 2)
	assert.Contains(t, ids, reopened.ID)
	assert.NotContains(t, ids, first.ID)

	got, err := repo.GetByID(f.ctx, repositories.GetAccountingDriftFindingRequest{
		TenantInfo: f.tenant,
		ID:         reopened.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, got.DifferenceMinor)
	assert.Equal(t, int64(-1_000), *got.DifferenceMinor)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err = repo.GetByID(f.ctx, repositories.GetAccountingDriftFindingRequest{
		TenantInfo: other,
		ID:         reopened.ID,
	})
	require.Error(t, err, "another tenant cannot read it")
}

func TestDriftFindingRepository_SummaryAttentionAndTable(t *testing.T) {
	f := setupSyncFixture(t)
	repo := NewDriftFindingRepository(DriftFindingParams{DB: f.conn, Logger: zap.NewNop()})

	day := int64(86_400)
	amountOld, err := repo.Create(f.ctx, f.driftFinding(accountingsync.DriftAmountMismatch, pulid.MustNew("inv_"), f.now-2*day))
	require.NoError(t, err)
	_, err = repo.Create(f.ctx, f.driftFinding(accountingsync.DriftAmountMismatch, pulid.MustNew("inv_"), f.now-3*day))
	require.NoError(t, err)
	_, err = repo.Create(f.ctx, f.driftFinding(accountingsync.DriftDeletedInProvider, pulid.MustNew("inv_"), f.now))
	require.NoError(t, err)
	resolved, err := repo.Create(f.ctx, f.driftFinding(accountingsync.DriftStatusMismatch, pulid.MustNew("inv_"), f.now-day))
	require.NoError(t, err)
	require.True(t, resolved.Clear(f.now))
	_, err = repo.Update(f.ctx, resolved)
	require.NoError(t, err)

	summary, err := repo.Summarize(f.ctx, &repositories.SummarizeAccountingDriftRequest{
		TenantInfo:    f.tenant,
		ConnectionID:  f.connection.ID,
		ResolvedSince: f.now - day,
	})
	require.NoError(t, err)
	assert.Equal(t, repositories.AccountingDriftSummary{
		Open:          3,
		AmountOpen:    2,
		GoneOpen:      1,
		ResolvedSince: 1,
	}, *summary)

	groups, err := repo.ListAttention(f.ctx, &repositories.ListAccountingDriftAttentionRequest{
		TenantInfo:     f.tenant,
		ConnectionID:   f.connection.ID,
		DetectedBefore: f.now - day,
	})
	require.NoError(t, err)
	require.Len(t, groups, 1, "only findings older than a day, and only open ones")
	assert.Equal(t, accountingsync.DriftAmountMismatch, groups[0].Kind)
	assert.Equal(t, 2, groups[0].Count)
	assert.Equal(t, f.now-3*day, groups[0].OldestDetectedAt)

	cursor, err := pagination.NewCursorInfo(10, "")
	require.NoError(t, err)
	cursor.IncludeTotalCount = true
	page, err := repo.ListConnection(f.ctx, &repositories.ListAccountingDriftFindingsConnectionRequest{
		Filter:       &pagination.QueryOptions{TenantInfo: f.tenant},
		Cursor:       cursor,
		ConnectionID: f.connection.ID,
		Statuses:     []accountingsync.DriftStatus{accountingsync.DriftStatusOpen},
		Kinds:        []accountingsync.DriftKind{accountingsync.DriftAmountMismatch},
	})
	require.NoError(t, err)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 2, *page.TotalCount)
	require.Len(t, page.Items, 2)
	assert.Equal(t, amountOld.ID, page.Items[0].ID, "newest detected first")
}

func TestDriftSource_ReadsTrenovaSideOfEverySyncedDocument(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	_, err := seeder.NewEngine(
		db,
		registry,
		&config.Config{System: config.SystemConfig{SystemUserPassword: "test-system-password"}},
	).Execute(ctx, seeder.ExecuteOptions{Environment: common.EnvDevelopment})
	require.NoError(t, err)

	var org candidateOrg
	require.NoError(t, db.NewSelect().Table("organizations").Column("id", "business_unit_id").Limit(1).Scan(ctx, &org))
	var userID pulid.ID
	require.NoError(t, db.NewSelect().Table("users").Column("id").
		Where("current_organization_id = ?", org.ID).Limit(1).Scan(ctx, &userID))
	shipments := make([]candidateShipment, 0, 3)
	require.NoError(t, db.NewSelect().Table("shipments").
		Column("id", "customer_id", "pro_number", "bol").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("id").Limit(3).Scan(ctx, &shipments))
	require.Len(t, shipments, 3)

	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	conn := postgres.NewTestConnection(db)
	logger := zap.NewNop()
	invoices := invoicerepository.New(invoicerepository.Params{DB: conn, Logger: logger})
	records := NewSyncRecordRepository(SyncRecordParams{DB: conn, Logger: logger})
	source := NewDriftSource(DriftSourceParams{DB: conn, Logger: logger})
	connection, err := NewConnectionRepository(ConnectionParams{DB: conn, Logger: logger}).
		Create(ctx, newConnection(tenant, userID, realm, 1_000))
	require.NoError(t, err)

	const dated = int64(1_790_000_000)
	posted := dated + 100

	createDoc := func(
		idx int,
		number string,
		billType billingqueue.BillType,
		status invoice.Status,
		total int64,
		applied int64,
		reference pulid.ID,
	) *invoice.Invoice {
		t.Helper()
		shp := shipments[idx]
		queue := &billingqueue.BillingQueueItem{
			OrganizationID:   org.ID,
			BusinessUnitID:   org.BusinessUnitID,
			ShipmentID:       shp.ID,
			BillToCustomerID: shp.CustomerID,
			Number:           number,
			Status:           billingqueue.StatusPosted,
			BillType:         billType,
		}
		_, insertErr := db.NewInsert().Model(queue).Exec(ctx)
		require.NoError(t, insertErr)
		created, createErr := invoices.Create(ctx, &invoice.Invoice{
			OrganizationID:     org.ID,
			BusinessUnitID:     org.BusinessUnitID,
			BillingQueueItemID: queue.ID,
			ShipmentID:         shp.ID,
			CustomerID:         shipments[0].CustomerID,
			Number:             number,
			BillType:           billType,
			Status:             status,
			PostedAt:           &posted,
			PaymentTerm:        invoice.PaymentTermNet30,
			CurrencyCode:       "USD",
			InvoiceDate:        dated,
			ShipmentProNumber:  shp.ProNumber,
			ShipmentBOL:        shp.BOL,
			BillToName:         "Test Customer",
			SubtotalAmount:     decimal.New(total, -2),
			OtherAmount:        decimal.Zero,
			TotalAmount:        decimal.New(total, -2),
			AppliedAmount:      decimal.New(applied, -2),
			SettlementStatus:   invoice.SettlementStatusUnpaid,
			DisputeStatus:      invoice.DisputeStatusNone,
			ReferenceInvoiceID: reference,
		})
		require.NoError(t, createErr)
		return created
	}
	synced := func(
		objectType accountingsync.SyncObjectType,
		objectID pulid.ID,
		operation accountingsync.SyncOperation,
		revision int64,
		externalID string,
	) *accountingsync.AccountingSyncRecord {
		t.Helper()
		record := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
			TenantInfo:   tenant,
			ConnectionID: connection.ID,
			Key: accountingsync.SyncRecordKey{
				ObjectType: objectType,
				ObjectID:   objectID,
				Operation:  operation,
				Revision:   revision,
			},
			SourceEvent:  accountingsync.SyncSourceInvoicePosted,
			DocumentDate: driftMinor(dated),
			At:           posted,
		})
		record.MarkSynced(&accountingsync.SyncResult{ExternalID: externalID}, posted)
		inserted, enqueueErr := records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{record})
		require.NoError(t, enqueueErr)
		require.Len(t, inserted.Inserted, 1)
		return inserted.Inserted[0]
	}

	open := createDoc(0, "INV-DRIFT-1", billingqueue.BillTypeInvoice, invoice.StatusPosted, 50_000, 10_000, pulid.Nil)
	voided := createDoc(1, "INV-DRIFT-2", billingqueue.BillTypeInvoice, invoice.StatusVoided, 20_000, 0, pulid.Nil)
	memo := createDoc(2, "CM-DRIFT-1", billingqueue.BillTypeCreditMemo, invoice.StatusPosted, -5_000, 0, open.ID)

	openCreate := synced(accountingsync.SyncObjectInvoice, open.ID, accountingsync.SyncOperationCreate, 1, "145")
	synced(accountingsync.SyncObjectInvoice, voided.ID, accountingsync.SyncOperationCreate, 1, "146")
	recreated := synced(accountingsync.SyncObjectInvoice, voided.ID, accountingsync.SyncOperationRecreate, 2, "147")

	memoRecord := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectCreditMemo,
			ObjectID:   memo.ID,
			Operation:  accountingsync.SyncOperationCreate,
		},
		SourceEvent:  accountingsync.SyncSourceCreditMemoPosted,
		DocumentDate: driftMinor(dated),
		At:           posted,
	})
	require.True(t, memoRecord.Reflect(userID, "145", "Made in Trenova to match; already there"))
	_, err = records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{memoRecord})
	require.NoError(t, err)

	sentMemo := createDoc(2, "DM-DRIFT-1", billingqueue.BillTypeDebitMemo, invoice.StatusPosted, 7_000, 0, open.ID)
	sentMemoRecord := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectDebitMemo,
			ObjectID:   sentMemo.ID,
			Operation:  accountingsync.SyncOperationCreate,
		},
		SourceEvent:  accountingsync.SyncSourceDebitMemoPosted,
		DocumentDate: driftMinor(dated),
		At:           posted,
	})
	sentMemoRecord.SetExternalRef(accountingsync.ExternalRefReflectedIn, "145")
	sentMemoRecord.MarkSynced(&accountingsync.SyncResult{ExternalID: "160"}, posted)
	_, err = records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{sentMemoRecord})
	require.NoError(t, err)

	listed, err := source.ListRecords(ctx, &repositories.ListAccountingDriftRecordsRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		DatedFrom:    dated,
		Limit:        10,
	})
	require.NoError(t, err)
	listedIDs := make([]pulid.ID, 0, len(listed))
	for _, record := range listed {
		listedIDs = append(listedIDs, record.ID)
	}
	assert.ElementsMatch(t, []pulid.ID{openCreate.ID, recreated.ID, sentMemoRecord.ID}, listedIDs,
		"the latest synced create per document, never a skipped memo or a superseded create")

	later, err := source.ListRecords(ctx, &repositories.ListAccountingDriftRecordsRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		DatedFrom:    dated + 1,
		Limit:        10,
	})
	require.NoError(t, err)
	assert.Empty(t, later, "documents dated before the scope are left out")

	states, err := source.ListStates(ctx, &repositories.ListAccountingDriftStatesRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		ObjectType:   accountingsync.SyncObjectInvoice,
		ObjectIDs:    []pulid.ID{open.ID, voided.ID},
	})
	require.NoError(t, err)
	byID := make(map[pulid.ID]*repositories.AccountingDriftState, len(states))
	for _, state := range states {
		byID[state.ObjectID] = state
	}
	require.Len(t, byID, 2)
	assert.Equal(t, "INV-DRIFT-1", byID[open.ID].Number)
	assert.Equal(t, shipments[0].CustomerID, byID[open.ID].PartyID)
	assert.NotEmpty(t, byID[open.ID].PartyName)
	assert.Equal(t, int64(50_000), byID[open.ID].AmountMinor)
	assert.Equal(t, int64(40_000), byID[open.ID].OpenMinor)
	assert.Equal(t, int64(-5_000), byID[open.ID].ReflectedMinor,
		"the reflected credit memo lowers what the provider should hold; a memo that was sent is its own document")
	assert.False(t, byID[open.ID].Voided)
	assert.True(t, byID[voided.ID].Voided)
	assert.Equal(t, "Voided", byID[voided.ID].State)
	assert.Zero(t, byID[voided.ID].ReflectedMinor)

	memoStates, err := source.ListStates(ctx, &repositories.ListAccountingDriftStatesRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		ObjectType:   accountingsync.SyncObjectCreditMemo,
		ObjectIDs:    []pulid.ID{memo.ID},
	})
	require.NoError(t, err)
	require.Len(t, memoStates, 1)
	assert.Equal(t, int64(5_000), memoStates[0].AmountMinor, "a credit memo compares by its size")

	for _, objectType := range accountingsync.DriftObjectTypes() {
		none, stateErr := source.ListStates(ctx, &repositories.ListAccountingDriftStatesRequest{
			TenantInfo:   tenant,
			ConnectionID: connection.ID,
			ObjectType:   objectType,
			ObjectIDs:    []pulid.ID{pulid.MustNew("obj_")},
		})
		require.NoError(t, stateErr, "%s states read", objectType)
		assert.Empty(t, none)
	}

	lines, err := source.ListBalances(ctx, &repositories.ListAccountingDriftBalancesRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		DatedFrom:    dated,
		Customers:    10,
	})
	require.NoError(t, err)
	require.Len(t, lines, 3, "the open and the voided invoice and the sent debit memo, all synced, for one customer")
	for _, line := range lines {
		assert.Equal(t, shipments[0].CustomerID, line.CustomerID)
		switch line.ObjectID {
		case open.ID:
			assert.Equal(t, int64(40_000), line.OpenMinor)
			assert.Equal(t, "145", line.ExternalID)
		case sentMemo.ID:
			assert.Equal(t, int64(7_000), line.OpenMinor)
		case voided.ID:
			assert.Zero(t, line.OpenMinor)
			assert.Equal(t, "147", line.ExternalID, "the recreated document's id")
		default:
			t.Fatalf("unexpected balance line %s", line.ObjectID)
		}
	}
	after, err := source.ListBalances(ctx, &repositories.ListAccountingDriftBalancesRequest{
		TenantInfo:      tenant,
		ConnectionID:    connection.ID,
		DatedFrom:       dated,
		AfterCustomerID: shipments[0].CustomerID,
		Customers:       10,
	})
	require.NoError(t, err)
	assert.Empty(t, after)

	pending, err := source.ListPendingCustomers(ctx, &repositories.ListAccountingDriftPendingCustomersRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		CustomerIDs:  []pulid.ID{shipments[0].CustomerID},
	})
	require.NoError(t, err)
	assert.Empty(t, pending)

	change := accountingsync.NewAccountingInboundChange(&accountingsync.InboundObservation{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		Kind:         accountingsync.InboundCustomerPayment,
		ExternalID:   "301",
		TxnDate:      dated,
		AmountMinor:  1_000,
		CurrencyCode: "USD",
		At:           posted,
	})
	change.PartyObjectID = shipments[0].CustomerID
	_, err = NewInboundChangeRepository(InboundChangeParams{DB: conn, Logger: logger}).Create(ctx, change)
	require.NoError(t, err)
	pending, err = source.ListPendingCustomers(ctx, &repositories.ListAccountingDriftPendingCustomersRequest{
		TenantInfo:   tenant,
		ConnectionID: connection.ID,
		CustomerIDs:  []pulid.ID{shipments[0].CustomerID},
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{shipments[0].CustomerID}, pending)

	start, err := source.ScopeStart(ctx, tenant)
	require.NoError(t, err)
	var earliest int64
	require.NoError(t, db.NewSelect().Table("fiscal_periods").
		ColumnExpr("COALESCE(MIN(start_date), 0)").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Where("status IN ('Open', 'Locked')").
		Scan(ctx, &earliest))
	assert.Equal(t, earliest, start)
}
