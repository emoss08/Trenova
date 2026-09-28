//go:build integration

package accountingsyncrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/driverpay"
	"github.com/emoss08/trenova/internal/core/domain/driversettlement"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type ledgerFixtureAccount struct {
	ID          pulid.ID `bun:"id"`
	AccountCode string   `bun:"account_code"`
}

type ledgerFixturePeriod struct {
	ID           pulid.ID `bun:"id"`
	FiscalYearID pulid.ID `bun:"fiscal_year_id"`
}

type ledgerFixtureLine struct {
	account  pulid.ID
	debit    int64
	credit   int64
	customer pulid.ID
}

type ledgerFixtureEntry struct {
	number     string
	entryType  journalentry.EntryType
	date       int64
	posted     bool
	reversalOf pulid.ID
	source     string
	sourceID   string
	lines      []ledgerFixtureLine
}

func TestLedgerSource_ReadsPostedJournalsWithTheirParties(t *testing.T) {
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
	var customer struct {
		ID   pulid.ID `bun:"id"`
		Name string   `bun:"name"`
	}
	require.NoError(t, db.NewSelect().Table("customers").Column("id", "name").
		Where("organization_id = ?", org.ID).Order("id").Limit(1).Scan(ctx, &customer))
	var carrier struct {
		ID   pulid.ID `bun:"id"`
		Name string   `bun:"name"`
	}
	require.NoError(t, db.NewSelect().Table("carriers").Column("id", "name").
		Where("organization_id = ?", org.ID).Order("id").Limit(1).Scan(ctx, &carrier))
	var driver struct {
		ID        pulid.ID `bun:"id"`
		FirstName string   `bun:"first_name"`
		LastName  string   `bun:"last_name"`
	}
	require.NoError(t, db.NewSelect().Table("workers").Column("id", "first_name", "last_name").
		Where("organization_id = ?", org.ID).Order("id").Limit(1).Scan(ctx, &driver))
	accounts := make([]ledgerFixtureAccount, 0, 3)
	require.NoError(t, db.NewSelect().Table("gl_accounts").Column("id", "account_code").
		Where("organization_id = ?", org.ID).
		Where("business_unit_id = ?", org.BusinessUnitID).
		Order("account_code").Limit(3).Scan(ctx, &accounts))
	require.Len(t, accounts, 3)
	acctA, acctB, acctC := accounts[0].ID, accounts[1].ID, accounts[2].ID

	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	

	const day = int64(1_900_000_000)
	const next = day + 86_400
	posted := day + 60

	conn := postgres.NewTestConnection(db)
	year, err := fiscalyearrepository.New(fiscalyearrepository.Params{DB: conn, Logger: zap.NewNop()}).
		Create(ctx, &fiscalyear.FiscalYear{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Status:         fiscalyear.StatusOpen,
			Year:           2030,
			Name:           "FY ledger",
			StartDate:      day - 30*86_400,
			EndDate:        day + 30*86_400,
		})
	require.NoError(t, err)
	fiscalPeriod, err := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: zap.NewNop()}).
		Create(ctx, &fiscalperiod.FiscalPeriod{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			FiscalYearID:   year.ID,
			PeriodNumber:   1,
			PeriodType:     fiscalperiod.PeriodTypeMonth,
			Status:         fiscalperiod.StatusOpen,
			Name:           "Ledger period",
			StartDate:      day - 30*86_400,
			EndDate:        day + 30*86_400,
		})
	require.NoError(t, err)
	period := ledgerFixturePeriod{ID: fiscalPeriod.ID, FiscalYearID: year.ID}
	source := NewLedgerSource(LedgerSourceParams{DB: conn, Logger: zap.NewNop()})

	carrierSettlement := &carriersettlement.CarrierSettlement{
		ID:               pulid.MustNew("carstl_"),
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		CarrierID:        carrier.ID,
		SettlementNumber: "CS-LEDGER-1",
		Status:           carriersettlement.StatusPosted,
		PeriodStart:      day - 7*86_400,
		PeriodEnd:        day,
		PayDate:          day + 15*86_400,
		NetPayableMinor:  1_000,
		CurrencyCode:     "USD",
		PostedAt:         &posted,
	}
	_, err = db.NewInsert().Model(carrierSettlement).Exec(ctx)
	require.NoError(t, err)
	driverSettlement := &driversettlement.Settlement{
		ID:               pulid.MustNew("dstl_"),
		OrganizationID:   org.ID,
		BusinessUnitID:   org.BusinessUnitID,
		WorkerID:         driver.ID,
		SettlementNumber: "DS-LEDGER-1",
		Status:           driversettlement.StatusPosted,
		Classification:   driverpay.PayeeClassificationOwnerOperator,
		PeriodStart:      day - 7*86_400,
		PeriodEnd:        day,
		PayDate:          day + 15*86_400,
		NetPayMinor:      300,
		CurrencyCode:     "USD",
		PostedAt:         &posted,
	}
	_, err = db.NewInsert().Model(driverSettlement).Exec(ctx)
	require.NoError(t, err)
	payment := &customerpayment.Payment{
		ID:                   pulid.MustNew("cpay_"),
		OrganizationID:       org.ID,
		BusinessUnitID:       org.BusinessUnitID,
		CustomerID:           customer.ID,
		PaymentDate:          day,
		AccountingDate:       day,
		AmountMinor:          500,
		UnappliedAmountMinor: 500,
		Status:               customerpayment.StatusPosted,
		PaymentMethod:        customerpayment.MethodCheck,
		ReferenceNumber:      "CHK-LEDGER-1",
		CurrencyCode:         "USD",
		CreatedByID:          userID,
	}
	_, err = db.NewInsert().Model(payment).Exec(ctx)
	require.NoError(t, err)

	batch := &journalentry.JournalBatch{
		ID:             pulid.MustNew("jb_"),
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
		BatchNumber:    "JB-LEDGER-1",
		BatchType:      "System",
		Status:         "Posted",
		Description:    "Ledger batch",
		AccountingDate: day,
		FiscalYearID:   period.FiscalYearID,
		FiscalPeriodID: period.ID,
		CreatedByID:    userID,
	}
	_, err = db.NewInsert().Model(batch).Exec(ctx)
	require.NoError(t, err)

	insert := func(fixture *ledgerFixtureEntry) pulid.ID {
		t.Helper()
		var total int64
		for _, line := range fixture.lines {
			total += line.debit
		}
		entry := &journalentry.JournalEntry{
			ID:             pulid.MustNew("je_"),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			FiscalYearID:   period.FiscalYearID,
			FiscalPeriodID: period.ID,
			BatchID:        batch.ID,
			EntryNumber:    fixture.number,
			EntryDate:      fixture.date,
			AccountingDate: fixture.date,
			EntryType:      fixture.entryType,
			Status:         journalentry.StatusPosted,
			Description:    fixture.number + " description",
			TotalDebit:     total,
			TotalCredit:    total,
			IsPosted:       fixture.posted,
			IsReversal:     !fixture.reversalOf.IsNil(),
			ReversalOfID:   fixture.reversalOf,
			CreatedByID:    userID,
		}
		if fixture.posted {
			entry.PostedAt = &posted
			entry.PostedByID = userID
		} else {
			entry.Status = journalentry.StatusPending
		}
		_, insertErr := db.NewInsert().Model(entry).Exec(ctx)
		require.NoError(t, insertErr)

		lines := make([]*journalentry.JournalEntryLine, 0, len(fixture.lines))
		for idx, line := range fixture.lines {
			lines = append(lines, &journalentry.JournalEntryLine{
				ID:             pulid.MustNew("jel_"),
				OrganizationID: org.ID,
				BusinessUnitID: org.BusinessUnitID,
				JournalEntryID: entry.ID,
				GLAccountID:    line.account,
				LineNumber:     int16(idx + 1),
				Description:    fixture.number + " line",
				DebitAmount:    line.debit,
				CreditAmount:   line.credit,
				NetAmount:      line.debit - line.credit,
				CustomerID:     line.customer,
			})
		}
		_, insertErr = db.NewInsert().Model(&lines).Exec(ctx)
		require.NoError(t, insertErr)

		if fixture.source != "" {
			_, insertErr = db.NewInsert().Model(&journalsource.Source{
				OrganizationID:       org.ID,
				BusinessUnitID:       org.BusinessUnitID,
				SourceObjectType:     fixture.source,
				SourceObjectID:       fixture.sourceID,
				SourceEventType:      "Posted",
				SourceDocumentNumber: fixture.number + "-DOC",
				Status:               "Posted",
				JournalBatchID:       batch.ID,
				JournalEntryID:       entry.ID,
			}).Exec(ctx)
			require.NoError(t, insertErr)
		}
		return entry.ID
	}

	settled := insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-1", entryType: journalentry.EntryTypeStandard, date: day, posted: true,
		source: journalsource.ObjectCarrierSettlement, sourceID: carrierSettlement.ID.String(),
		lines: []ledgerFixtureLine{{account: acctB, debit: 1_000}, {account: acctA, credit: 1_000}},
	})
	paid := insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-2", entryType: journalentry.EntryTypeStandard, date: day, posted: true,
		source: journalsource.ObjectCustomerPayment, sourceID: payment.ID.String(),
		lines: []ledgerFixtureLine{
			{account: acctB, debit: 500},
			{account: acctA, credit: 500, customer: customer.ID},
		},
	})
	reversed := insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-3", entryType: journalentry.EntryTypeReversal, date: day, posted: true,
		reversalOf: settled, source: journalsource.ObjectJournalEntry, sourceID: settled.String(),
		lines: []ledgerFixtureLine{{account: acctA, debit: 1_000}, {account: acctB, credit: 1_000}},
	})
	insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-4", entryType: journalentry.EntryTypeClosing, date: day, posted: true,
		source: journalsource.ObjectFiscalYear, sourceID: period.FiscalYearID.String(),
		lines: []ledgerFixtureLine{{account: acctC, debit: 9_000}, {account: acctB, credit: 9_000}},
	})
	pending := insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-5", entryType: journalentry.EntryTypeStandard, date: day, posted: false,
		lines: []ledgerFixtureLine{{account: acctC, debit: 7_000}, {account: acctA, credit: 7_000}},
	})
	driven := insert(&ledgerFixtureEntry{
		number: "JE-LEDGER-6", entryType: journalentry.EntryTypeStandard, date: next, posted: true,
		source: journalsource.ObjectDriverSettlement, sourceID: driverSettlement.ID.String(),
		lines: []ledgerFixtureLine{{account: acctC, debit: 300}, {account: acctA, credit: 300}},
	})

	carrierParty := repositories.LedgerParty{
		Kind: repositories.LedgerPartyCarrier, ID: carrier.ID, Name: carrier.Name,
	}
	customerParty := repositories.LedgerParty{
		Kind: repositories.LedgerPartyCustomer, ID: customer.ID, Name: customer.Name,
	}
	driverParty := repositories.LedgerParty{
		Kind: repositories.LedgerPartyDriver, ID: driver.ID, Name: driver.FirstName + " " + driver.LastName,
	}

	t.Run("a reversal carries the party and number of the entry it reverses", func(t *testing.T) {
		journal, getErr := source.GetJournal(ctx, &repositories.GetLedgerJournalRequest{TenantInfo: tenant, ID: reversed})
		require.NoError(t, getErr)
		assert.True(t, journal.IsReversal)
		assert.Equal(t, "JE-LEDGER-1", journal.ReversalOfNumber)
		assert.Equal(t, journalsource.ObjectCarrierSettlement, journal.SourceObjectType)
		assert.Equal(t, carrierSettlement.ID.String(), journal.SourceObjectID)
		assert.Equal(t, "JE-LEDGER-1-DOC", journal.SourceDocumentNumber)
		assert.Equal(t, carrierParty, journal.Party)
		assert.Equal(t, day, journal.AccountingDate)
		assert.Equal(t, posted, journal.PostedAt)
		require.Len(t, journal.Lines, 2)
		assert.Equal(t, acctA, journal.Lines[0].AccountID)
		assert.Equal(t, accounts[0].AccountCode, journal.Lines[0].AccountCode)
		assert.Equal(t, int64(1_000), journal.Lines[0].DebitMinor)
		assert.Equal(t, int64(1_000), journal.Lines[1].CreditMinor)
	})

	t.Run("an unposted entry or another tenant's is not found", func(t *testing.T) {
		_, getErr := source.GetJournal(ctx, &repositories.GetLedgerJournalRequest{TenantInfo: tenant, ID: pending})
		assert.True(t, errortypes.IsNotFoundError(getErr))
		_, getErr = source.GetJournal(ctx, &repositories.GetLedgerJournalRequest{TenantInfo: other, ID: paid})
		assert.True(t, errortypes.IsNotFoundError(getErr))
	})

	t.Run("a day lists its posted entries, never closing ones", func(t *testing.T) {
		journals, listErr := source.ListJournals(ctx, &repositories.ListLedgerJournalsRequest{
			TenantInfo: tenant, From: day, Before: next,
		})
		require.NoError(t, listErr)
		numbers := make([]string, 0, len(journals))
		for _, journal := range journals {
			numbers = append(numbers, journal.EntryNumber)
		}
		assert.Equal(t, []string{"JE-LEDGER-1", "JE-LEDGER-2", "JE-LEDGER-3"}, numbers)
		assert.Equal(t, customerParty, journals[1].Party)
		assert.Equal(t, customerParty, journals[1].Lines[1].Party)
		assert.True(t, journals[1].Lines[0].Party.IsZero())

		journals, listErr = source.ListJournals(ctx, &repositories.ListLedgerJournalsRequest{
			TenantInfo: other, From: day, Before: next,
		})
		require.NoError(t, listErr)
		assert.Empty(t, journals)
	})

	t.Run("sums split party accounts by who the line belongs to", func(t *testing.T) {
		from := day
		balances, sumErr := source.SumLines(ctx, &repositories.SumLedgerRequest{
			TenantInfo: tenant, From: &from, Before: next + 86_400, PartyAccountIDs: []pulid.ID{acctA},
		})
		require.NoError(t, sumErr)
		type sum struct {
			account pulid.ID
			party   repositories.LedgerParty
			debit   int64
			credit  int64
		}
		got := make([]sum, 0, len(balances))
		for idx := range balances {
			got = append(got, sum{
				account: balances[idx].AccountID,
				party:   balances[idx].Party,
				debit:   balances[idx].DebitMinor,
				credit:  balances[idx].CreditMinor,
			})
			assert.NotEmpty(t, balances[idx].Category)
		}
		assert.ElementsMatch(t, []sum{
			{account: acctA, party: carrierParty, debit: 1_000, credit: 1_000},
			{account: acctA, party: customerParty, credit: 500},
			{account: acctA, party: driverParty, credit: 300},
			{account: acctB, debit: 1_500, credit: 1_000},
			{account: acctC, debit: 300},
		}, got)

		balances, sumErr = source.SumLines(ctx, &repositories.SumLedgerRequest{
			TenantInfo: tenant, From: &from, Before: next + 86_400,
		})
		require.NoError(t, sumErr)
		require.Len(t, balances, 3)
		assert.Equal(t, acctA, balances[0].AccountID)
		assert.True(t, balances[0].Party.IsZero())
		assert.Equal(t, int64(-800), balances[0].NetMinor())

		balances, sumErr = source.SumLines(ctx, &repositories.SumLedgerRequest{
			TenantInfo: tenant, From: &from, Before: next + 86_400, IncludeClosing: true,
		})
		require.NoError(t, sumErr)
		require.Len(t, balances, 3)
		assert.Equal(t, int64(-800), balances[0].NetMinor())
		assert.Equal(t, int64(500-9_000), balances[1].NetMinor())
		assert.Equal(t, int64(9_300), balances[2].NetMinor())

		later := next
		balances, sumErr = source.SumLines(ctx, &repositories.SumLedgerRequest{
			TenantInfo: tenant, From: &later, Before: next + 86_400,
		})
		require.NoError(t, sumErr)
		require.Len(t, balances, 2)
		assert.Equal(t, int64(-300), balances[0].NetMinor())
		assert.Equal(t, int64(300), balances[1].NetMinor())
	})

	t.Run("active accounts are those with posted lines since the date", func(t *testing.T) {
		since := next
		active, activeErr := source.ListActiveAccounts(ctx, &repositories.ListLedgerAccountsRequest{
			TenantInfo: tenant, Since: &since,
		})
		require.NoError(t, activeErr)
		ids := make([]pulid.ID, 0, len(active))
		for idx := range active {
			ids = append(ids, active[idx].ID)
		}
		assert.Equal(t, []pulid.ID{acctA, acctC}, ids)

		active, activeErr = source.ListActiveAccounts(ctx, &repositories.ListLedgerAccountsRequest{TenantInfo: tenant})
		require.NoError(t, activeErr)
		ids = ids[:0]
		for idx := range active {
			ids = append(ids, active[idx].ID)
		}
		assert.Subset(t, ids, []pulid.ID{acctA, acctB, acctC})
	})

	t.Run("posted journals without a record are candidates", func(t *testing.T) {
		records := NewSyncRecordRepository(SyncRecordParams{DB: conn, Logger: zap.NewNop()})
		connection, connErr := NewConnectionRepository(ConnectionParams{DB: conn, Logger: zap.NewNop()}).
			Create(ctx, newConnection(tenant, userID, realm, 1_000))
		require.NoError(t, connErr)
		list := func(objectType accountingsync.SyncObjectType) []pulid.ID {
			t.Helper()
			found, listErr := records.ListCandidates(ctx, &repositories.ListAccountingSyncCandidatesRequest{
				TenantInfo:   tenant,
				ConnectionID: connection.ID,
				ObjectType:   objectType,
				Operation:    accountingsync.SyncOperationCreate,
				DatedFrom:    day - 86_400,
				Timezone:     "UTC",
				Limit:        50,
			})
			require.NoError(t, listErr)
			ids := make([]pulid.ID, 0, len(found))
			for idx := range found {
				ids = append(ids, found[idx].ObjectID)
			}
			return ids
		}
		queue := func(objectType accountingsync.SyncObjectType, objectID pulid.ID, at int64) {
			t.Helper()
			_, enqueueErr := records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{
				accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
					TenantInfo:   tenant,
					ConnectionID: connection.ID,
					Key: accountingsync.SyncRecordKey{
						ObjectType: objectType,
						ObjectID:   objectID,
						Operation:  accountingsync.SyncOperationCreate,
						Revision:   1,
					},
					SourceEvent: accountingsync.SyncSourceJournalPosted,
					At:          at,
				}),
			})
			require.NoError(t, enqueueErr)
		}

		assert.ElementsMatch(t, []pulid.ID{settled, paid, reversed, driven},
			list(accountingsync.SyncObjectJournalEntry),
			"closing and unposted entries are never candidates")
		queue(accountingsync.SyncObjectJournalEntry, paid, posted)
		assert.ElementsMatch(t, []pulid.ID{settled, reversed, driven},
			list(accountingsync.SyncObjectJournalEntry))

		assert.ElementsMatch(t, []pulid.ID{settled, paid, reversed, driven},
			list(accountingsync.SyncObjectJournalSummary))
		queue(accountingsync.SyncObjectJournalSummary,
			pulid.ID(accountingsync.JournalDayID(day, time.UTC)), posted+1_000)
		queue(accountingsync.SyncObjectJournalSummary,
			pulid.ID(accountingsync.JournalDayID(next, time.UTC)), posted-1_000)
		assert.Equal(t, []pulid.ID{driven}, list(accountingsync.SyncObjectJournalSummary),
			"a day queued after its entries were posted covers them; one queued before does not")
	})
}
