//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"fmt"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype/accounttypedb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod/fiscalperioddb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal/journalreversaldb"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal/manualjournaldb"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/glaccountrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalentryrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalreversalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalsourcerepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/manualjournalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalcardservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accounttype"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/accountingcontrolrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/customerrepository"
	"github.com/emoss08/trenova/internal/modules/places/adapters/postgres/locationrepository"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const ledgerRowsPerTenant = 5

// ledgerBooks is one tenant's books: the period its rows post to, and the
// rows each ledger list reads, with the ones its typed filter selects.
type ledgerBooks struct {
	tenant *agentreadtest.Tenant
	ctx    context.Context
	rows   map[string][]string
	marked map[string][]string
	entry  string
	manual string
}

func ledgerResolvers(conn *postgres.Connection) agentread.ResolverParams {
	logger := zap.NewNop()
	entries := journalentryrepository.New(journalentryrepository.Params{DB: conn, Logger: logger})
	periods := fiscalperiodrepository.New(fiscalperiodrepository.Params{DB: conn, Logger: logger})
	journals := manualjournalrepository.New(manualjournalrepository.Params{DB: conn, Logger: logger})
	reversals := journalreversalrepository.New(
		journalreversalrepository.Params{DB: conn, Logger: logger})
	sources := journalsourcerepository.New(journalsourcerepository.Params{DB: conn, Logger: logger})
	ledger := journalentryservice.New(journalentryservice.Params{
		Logger: logger, EntryRepo: entries, SourceRepo: sources,
	})

	return agentread.ResolverParams{
		JournalEntryService: ledger,
		FiscalPeriodService: fiscalperiodservice.New(fiscalperiodservice.Params{
			Logger: logger, Repo: periods,
		}),
		ManualJournalService: manualjournalservice.New(manualjournalservice.Params{
			Logger: logger, Repo: journals,
		}),
		JournalReversalService: journalreversalservice.New(journalreversalservice.Params{
			Logger: logger, JournalReversalRepo: reversals,
		}),
		JournalCards: journalcardservice.New(journalcardservice.Params{
			Logger:    logger,
			Entries:   entries,
			Journals:  journals,
			Sources:   sources,
			Reversals: reversals,
			Accounts:  glaccountrepository.New(glaccountrepository.Params{DB: conn, Logger: logger}),
			Periods:   periods,
			Controls: accountingcontrolrepository.New(
				accountingcontrolrepository.Params{DB: conn, Logger: logger}),
			Users: userrepository.New(userrepository.Params{DB: conn, Logger: logger}),
			Customers: customerrepository.New(customerrepository.Params{
				DB: conn, Logger: logger,
			}),
			Locations: locationrepository.New(locationrepository.Params{DB: conn, Logger: logger}),
			Ledger:    ledger,
		}),
	}
}

// writeBooks writes a tenant's books through the admin connection: a
// calendar year's periods, five journal entries (the last one posted), five
// manual journals and five reversals (the last of each approved), and one
// line on the first entry and the first journal so a card has an account.
//
//nolint:funlen // one tenant's books, written in the order the keys need
func writeBooks(t *testing.T, admin *keel.DB, tenant *agentreadtest.Tenant, label string) *ledgerBooks {
	t.Helper()

	ctx := t.Context()
	org, bu, user := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID, tenant.Actor.UserID
	day := time.Date(2026, time.March, 16, 12, 0, 0, 0, time.UTC).Unix()
	fy, fp := fiscaltest.OpenPeriod(t, ctx, fiscaltest.OpenPeriodParams{
		DB: admin, OrganizationID: org, BusinessUnitID: bu, At: day,
	})
	books := &ledgerBooks{
		tenant: tenant,
		ctx:    readscope.With(dbscope.WithTenant(ctx, tenant.Scope()), nil),
		rows:   make(map[string][]string),
		marked: make(map[string][]string),
	}

	writeCtx := tenant.Ctx(ctx)
	must := func(err error) {
		t.Helper()
		require.NoError(t, err)
	}
	periods, err := fiscalperioddb.FiscalPeriods.Query().All(writeCtx, admin)
	require.NoError(t, err)
	for _, period := range periods {
		books.rows[toolLedgerPeriods] = append(books.rows[toolLedgerPeriods], period.ID.String())
		if period.Status == fiscalperiod.StatusOpen {
			books.marked[toolLedgerPeriods] = append(books.marked[toolLedgerPeriods],
				period.ID.String())
		}
	}

	_, err = keel.Exec(ctx, admin,
		"INSERT INTO accounting_controls (id, organization_id, business_unit_id) VALUES ($1, $2, $3)",
		pulid.MustNew("ac_").String(), org.String(), bu.String())
	require.NoError(t, err)

	kind := &accounttype.AccountType{
		ID: pulid.MustNew("at_"), OrganizationID: org, BusinessUnitID: bu,
		Code: "EXP" + label, Name: "Expense " + label, Category: accounttype.CategoryExpense,
		Status: domaintypes.StatusActive,
	}
	must(accounttypedb.AccountTypes.Create(writeCtx, admin, kind))
	account := &glaccount.GLAccount{
		ID: pulid.MustNew("gla_"), OrganizationID: org, BusinessUnitID: bu,
		AccountTypeID: kind.ID, AccountCode: "5100" + label, Name: "Fuel expense " + label,
		Status: domaintypes.StatusActive,
	}
	must(glaccountdb.GLAccounts.Create(writeCtx, admin, account))
	batch := &journalentry.JournalBatch{
		ID: pulid.MustNew("jb_"), OrganizationID: org, BusinessUnitID: bu,
		BatchNumber: "JB-" + label, BatchType: "System", Status: "Posted", Description: "Batch",
		AccountingDate: day, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID, CreatedByID: user,
	}
	must(journalentrydb.JournalBatches.Create(writeCtx, admin, batch))

	for idx := range ledgerRowsPerTenant {
		marked := idx == ledgerRowsPerTenant-1
		status := journalentry.StatusPending
		if marked {
			status = journalentry.StatusPosted
		}
		entry := &journalentry.JournalEntry{
			ID: pulid.MustNew("je_"), OrganizationID: org, BusinessUnitID: bu,
			BatchID: batch.ID, FiscalYearID: fy.ID, FiscalPeriodID: fp.ID,
			EntryNumber: fmt.Sprintf("JE-%s%d", label, idx), EntryDate: day,
			AccountingDate: day + int64(idx), EntryType: journalentry.EntryTypeStandard,
			Status: status, ReferenceType: "Invoice", Description: "Fuel " + label,
			TotalDebit: 5000, TotalCredit: 5000, ReversalReason: "Booked to the wrong period",
			CreatedByID: user,
		}
		must(journalentrydb.JournalEntries.Create(writeCtx, admin, entry))
		books.rows[toolLedgerEntries] = append(books.rows[toolLedgerEntries], entry.ID.String())
		if marked {
			books.marked[toolLedgerEntries] = append(books.marked[toolLedgerEntries],
				entry.ID.String())
		}
		if idx == 0 {
			books.entry = entry.ID.String()
			must(journalentrydb.JournalEntryLines.Create(writeCtx, admin, &journalentry.JournalEntryLine{
				ID: pulid.MustNew("jel_"), OrganizationID: org, BusinessUnitID: bu,
				JournalEntryID: entry.ID, GLAccountID: account.ID, LineNumber: 1,
				Description: "Diesel", DebitAmount: 5000, NetAmount: 5000,
			}))
		}

		journalStatus := manualjournal.StatusDraft
		if marked {
			journalStatus = manualjournal.StatusApproved
		}
		journal := &manualjournal.Request{
			ID: pulid.MustNew("mjr_"), OrganizationID: org, BusinessUnitID: bu,
			RequestNumber: fmt.Sprintf("MJ-%s%d", label, idx), Status: journalStatus,
			Description: "Accrual " + label, Reason: "Month end", AccountingDate: day,
			RequestedFiscalYearID: fy.ID, RequestedFiscalPeriodID: fp.ID, CurrencyCode: "USD",
			TotalDebit: 900, TotalCredit: 900, CreatedByID: user,
		}
		must(manualjournaldb.Requests.Create(writeCtx, admin, journal))
		books.rows[toolLedgerJournals] = append(books.rows[toolLedgerJournals], journal.ID.String())
		if marked {
			books.marked[toolLedgerJournals] = append(books.marked[toolLedgerJournals],
				journal.ID.String())
		}
		if idx == 0 {
			books.manual = journal.ID.String()
			must(manualjournaldb.Lines.Create(writeCtx, admin, &manualjournal.Line{
				ID: pulid.MustNew("mjrl_"), OrganizationID: org, BusinessUnitID: bu,
				ManualJournalRequestID: journal.ID, LineNumber: 1, GLAccountID: account.ID,
				Description: "Accrued fuel", DebitAmount: 900,
			}))
		}

		reversalStatus := journalreversal.StatusRequested
		if marked {
			reversalStatus = journalreversal.StatusApproved
		}
		reversal := &journalreversal.Reversal{
			ID: pulid.MustNew("jrev_"), OrganizationID: org, BusinessUnitID: bu,
			OriginalJournalEntryID: entry.ID, Status: reversalStatus,
			RequestedAccountingDate: day, ResolvedFiscalYearID: fy.ID, ResolvedFiscalPeriodID: fp.ID,
			ReasonCode: "WrongPeriod", ReasonText: "Booked to February", RequestedByID: user,
		}
		must(journalreversaldb.Reversals.Create(writeCtx, admin, reversal))
		books.rows[toolLedgerReversals] = append(books.rows[toolLedgerReversals],
			reversal.ID.String())
		if marked {
			books.marked[toolLedgerReversals] = append(books.marked[toolLedgerReversals],
				reversal.ID.String())
		}
	}

	return books
}

// ledgerMarked is each ledger list's typed filter, selecting the rows the
// books mark.
var ledgerMarked = map[string]map[string]any{
	toolLedgerEntries:   {"status": "Posted"},
	toolLedgerPeriods:   {"status": []any{"Open"}},
	toolLedgerJournals:  {"status": []any{"Approved"}},
	toolLedgerReversals: {"status": []any{"Approved"}},
}

func atCeiling(
	t *testing.T,
	ctx context.Context,
	tool agentports.AgentQueryTool,
	actor *agentreadtest.Tenant,
	args map[string]any,
	ceiling permission.FieldSensitivity,
) (map[string]any, error) {
	t.Helper()

	data, err := tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID:    actor.Actor.OrganizationID,
		BusinessUnitID:    actor.Actor.BusinessUnitID,
		Actor:             actor.Actor,
		Timezone:          "America/Chicago",
		Params:            args,
		DataAccessCeiling: ceiling,
		RunID:             pulid.MustNew("ar_"),
	})
	if err != nil {
		return nil, err
	}

	return asMap(t, data), nil
}

func TestConvertedLedgerReads_ReadTheTenantsBooksUnderRowLevelSecurity(t *testing.T) {
	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	services := ledgerResolvers(conn)
	engine := &mocks.DenyResourcesPermissionEngine{}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{Engine: engine, Resolvers: services})
	public := agentreadtest.NewReader(t, agentreadtest.Options{Engine: &publicRole{}, Resolvers: services})
	accountant := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: &restrictedRole{}, Resolvers: services,
	})

	ours := writeBooks(t, admin, agentreadtest.SeedTenant(t, admin, "A"), "A")
	theirs := writeBooks(t, admin, agentreadtest.SeedTenant(t, admin, "B"), "B")

	for name, filter := range ledgerMarked {
		t.Run(name, func(t *testing.T) {
			converted, err := agentquerytoolservice.ConvertedListTool(name, reader, engine)
			require.NoError(t, err)
			actor := ours.tenant.Actor

			t.Run("every page once, and nothing of another tenant", func(t *testing.T) {
				ids, pages := walk(t, ours.ctx, converted, actor, nil)
				assert.Greater(t, pages, 1, "the walk must cross a page")
				assert.ElementsMatch(t, ours.rows[name], ids)
				for _, id := range theirs.rows[name] {
					assert.NotContains(t, ids, id)
				}
			})

			t.Run("a typed filter narrows to what it names", func(t *testing.T) {
				ids, _ := walk(t, ours.ctx, converted, actor, filter)
				assert.ElementsMatch(t, ours.marked[name], ids)
			})

			t.Run("what the repository tool withheld stays withheld at Internal", func(t *testing.T) {
				page, err := atCeiling(t, ours.ctx, converted, ours.tenant,
					map[string]any{"limit": 50}, permission.SensitivityInternal)
				require.NoError(t, err)
				items, _ := page["items"].([]any)
				require.Len(t, items, len(ours.rows[name]))
				for _, item := range items {
					row, _ := item.(map[string]any)
					for _, field := range oldGated[name] {
						assert.NotContains(t, row, field)
					}
				}
				for _, field := range oldGated[name] {
					assert.Contains(t, page["withheldByAccess"], field)
				}
			})

			t.Run("a field the role cannot see stays withheld", func(t *testing.T) {
				hidden, err := agentquerytoolservice.ConvertedListTool(name, public, &publicRole{})
				require.NoError(t, err)
				page := callTool(t, ours.ctx, hidden, actor, map[string]any{"limit": 50})
				withheld, _ := page["withheldByAccess"].([]any)
				require.NotEmpty(t, withheld)
				items, _ := page["items"].([]any)
				require.Len(t, items, len(ours.rows[name]))
				for _, item := range items {
					row, _ := item.(map[string]any)
					assert.Contains(t, row, "id")
					for _, field := range withheld {
						assert.NotContains(t, row, field)
					}
				}
			})
		})
	}

	cards := map[string]struct {
		id       string
		other    string
		record   string
		withheld []string
	}{
		"ledger_get_journal_entry": {
			id: ours.entry, other: theirs.entry, record: "entry",
			withheld: []string{
				"entry.totalDebit", "entry.totalCredit", "entry.reversalReason",
				"lines.debitAmount", "lines.creditAmount",
			},
		},
		"ledger_get_manual_journal": {
			id: ours.manual, other: theirs.manual, record: "journal",
			withheld: []string{
				"journal.totalDebit", "journal.totalCredit", "journal.reason",
				"lines.debitAmount", "lines.creditAmount",
			},
		},
	}
	for name, c := range cards {
		t.Run(name, func(t *testing.T) {
			get, err := agentquerytoolservice.ConvertedGetTool(name, reader, engine)
			require.NoError(t, err)
			param := get.ParamSchema()["required"].([]string)[0]

			card, err := atCeiling(t, ours.ctx, get, ours.tenant, map[string]any{param: c.id},
				permission.SensitivityInternal)
			require.NoError(t, err)
			record, _ := card[c.record].(map[string]any)
			assert.Equal(t, c.id, record["id"])
			lines, _ := card["lines"].([]any)
			require.Len(t, lines, 1)
			line, _ := lines[0].(map[string]any)
			assert.Contains(t, line, "accountName", "the account a line posts to is shown")
			assert.Contains(t, line, "accountCode")
			assert.NotContains(t, line, "debitAmount", "a line's amount is withheld at Internal")
			assert.NotContains(t, record, "totalDebit")
			for _, path := range c.withheld {
				assert.Contains(t, card["withheldByAccess"], path, name)
			}

			reaching, err := agentquerytoolservice.ConvertedGetTool(name, accountant, &restrictedRole{})
			require.NoError(t, err)
			whole, err := atCeiling(t, ours.ctx, reaching, ours.tenant, map[string]any{param: c.id},
				permission.SensitivityRestricted)
			require.NoError(t, err)
			wholeLines, _ := whole["lines"].([]any)
			require.Len(t, wholeLines, 1)
			assert.Contains(t, wholeLines[0], "debitAmount", "Restricted reaches the amounts")

			_, err = atCeiling(t, ours.ctx, get, ours.tenant, map[string]any{param: c.other},
				permission.SensitivityInternal)
			require.Error(t, err, "another tenant's record is not found")
		})
	}
}
