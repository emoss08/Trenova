package agentquerytoolservice_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/pagetest"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalreversal"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalreversalports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	ledgerAccountingDay = int64(1_773_964_800)
	ledgerZone          = "America/Chicago"
)

// ledgerFilters records the table filter each ledger list's resolver
// receives, and answers with one fully written row.
type ledgerFilters struct {
	mu      sync.Mutex
	filters map[string]*pagination.QueryOptions
}

func (l *ledgerFilters) record(name string, filter *pagination.QueryOptions) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.filters[name] = filter
}

func (l *ledgerFilters) last(name string) *pagination.QueryOptions {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.filters[name]
}

type ledgerEntries struct {
	journalentryports.JournalEntryRepository

	seen *ledgerFilters
}

func (r *ledgerEntries) ListConnection(
	_ context.Context,
	req *journalentryports.ListJournalEntryConnectionRequest,
) (*pagination.CursorListResult[*journalentry.JournalEntry], error) {
	r.seen.record(toolLedgerEntries, req.Filter)

	return pagetest.Of([]*journalentry.JournalEntry{{
		ID: pulid.MustNew("je_"), EntryNumber: "JE-7", EntryDate: ledgerAccountingDay,
		AccountingDate: ledgerAccountingDay, EntryType: journalentry.EntryTypeStandard,
		Status: journalentry.StatusPosted, Description: "Fuel", ReferenceType: "Invoice",
		ReferenceNumber: "INV-1", FiscalPeriodID: pulid.MustNew("fp_"),
		TotalDebit: 5000, TotalCredit: 5000, ReversalReason: "Wrong period",
		RejectionReason: "Wrong account", ApprovalNotes: "Checked",
		CreatedAt: ledgerAccountingDay,
	}}, req.Cursor.Limit), nil
}

type ledgerPeriods struct {
	fiscalperiodports.FiscalPeriodRepository

	seen *ledgerFilters
}

func (r *ledgerPeriods) ListConnection(
	_ context.Context,
	req *fiscalperiodports.ListFiscalPeriodConnectionRequest,
) (*pagination.CursorListResult[*fiscalperiod.FiscalPeriod], error) {
	r.seen.record(toolLedgerPeriods, req.Filter)
	reopened := ledgerAccountingDay

	return pagetest.Of([]*fiscalperiod.FiscalPeriod{{
		ID: pulid.MustNew("fp_"), FiscalYearID: pulid.MustNew("fy_"), Name: "March 2026",
		PeriodNumber: 3, PeriodType: fiscalperiod.PeriodTypeMonth,
		Status: fiscalperiod.StatusOpen, StartDate: ledgerAccountingDay,
		EndDate: ledgerAccountingDay + 86_400*30, ReopenedAt: &reopened,
		ReopenReason: "Late vendor bill", CreatedAt: ledgerAccountingDay,
	}}, req.Cursor.Limit), nil
}

type ledgerJournals struct {
	manualjournalports.ManualJournalRepository

	seen *ledgerFilters
}

func (r *ledgerJournals) ListConnection(
	_ context.Context,
	req *manualjournalports.ListManualJournalConnectionRequest,
) (*pagination.CursorListResult[*manualjournal.Request], error) {
	r.seen.record(toolLedgerJournals, req.Filter)

	return pagetest.Of([]*manualjournal.Request{{
		ID: pulid.MustNew("mjr_"), RequestNumber: "MJ-3", Status: manualjournal.StatusApproved,
		Description: "Accrue fuel", Reason: "Month end accrual", CurrencyCode: "USD",
		AccountingDate: ledgerAccountingDay, RequestedFiscalPeriodID: pulid.MustNew("fp_"),
		TotalDebit: 900, TotalCredit: 900, RejectionReason: "Too early",
		CancelReason: "Duplicate", CreatedAt: ledgerAccountingDay,
	}}, req.Cursor.Limit), nil
}

type ledgerReversals struct {
	journalreversalports.JournalReversalRepository

	seen *ledgerFilters
}

func (r *ledgerReversals) ListConnection(
	_ context.Context,
	req *journalreversalports.ListJournalReversalConnectionRequest,
) (*pagination.CursorListResult[*journalreversal.Reversal], error) {
	r.seen.record(toolLedgerReversals, req.Filter)

	return pagetest.Of([]*journalreversal.Reversal{{
		ID: pulid.MustNew("jrev_"), Status: journalreversal.StatusApproved,
		OriginalJournalEntryID: pulid.MustNew("je_"), RequestedAccountingDate: ledgerAccountingDay,
		ResolvedFiscalPeriodID: pulid.MustNew("fp_"), ReasonCode: "WrongPeriod",
		ReasonText: "Booked to February", RejectionReason: "Not needed",
		CancelReason: "Withdrawn", CreatedAt: ledgerAccountingDay,
	}}, req.Cursor.Limit), nil
}

const (
	toolLedgerEntries   = "ledger_list_journal_entries"
	toolLedgerPeriods   = "ledger_list_fiscal_periods"
	toolLedgerJournals  = "ledger_list_manual_journals"
	toolLedgerReversals = "ledger_list_journal_reversals"
)

func ledgerServices(seen *ledgerFilters) agentread.ResolverParams {
	logger := zap.NewNop()

	return agentread.ResolverParams{
		JournalEntryService: journalentryservice.New(journalentryservice.Params{
			Logger: logger, EntryRepo: &ledgerEntries{seen: seen},
		}),
		FiscalPeriodService: fiscalperiodservice.New(fiscalperiodservice.Params{
			Logger: logger, Repo: &ledgerPeriods{seen: seen},
		}),
		ManualJournalService: manualjournalservice.New(manualjournalservice.Params{
			Logger: logger, Repo: &ledgerJournals{seen: seen},
		}),
		JournalReversalService: journalreversalservice.New(journalreversalservice.Params{
			Logger: logger, JournalReversalRepo: &ledgerReversals{seen: seen},
		}),
	}
}

func queryLedger(
	t *testing.T,
	tool string,
	args map[string]any,
	ceiling permission.FieldSensitivity,
) (map[string]any, *ledgerFilters, error) {
	t.Helper()

	seen := &ledgerFilters{filters: make(map[string]*pagination.QueryOptions)}
	var engine permissionports.PermissionEngine = &mocks.DenyResourcesPermissionEngine{}
	if ceiling == permission.SensitivityRestricted {
		engine = &restrictedRole{}
	}
	reader := agentreadtest.NewReader(t, agentreadtest.Options{
		Engine: engine, Resolvers: ledgerServices(seen),
	})
	converted, err := agentquerytoolservice.ConvertedListTool(tool, reader, engine)
	require.NoError(t, err)
	person := agentreadtest.Person()
	data, err := converted.Query(readscope.With(t.Context(), nil), &agentports.QueryToolParams{
		OrganizationID:    person.OrganizationID,
		BusinessUnitID:    person.BusinessUnitID,
		Actor:             person,
		Timezone:          ledgerZone,
		Params:            args,
		DataAccessCeiling: ceiling,
		RunID:             pulid.MustNew("ar_"),
	})
	if err != nil {
		return nil, seen, err
	}

	return asMap(t, data), seen, nil
}

// restrictedRole is a person whose role reaches Restricted fields, as an
// accountant's does.
type restrictedRole struct {
	mocks.DenyResourcesPermissionEngine
}

func (restrictedRole) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*permissionports.ResourcePermissionDetail, error) {
	return &permissionports.ResourcePermissionDetail{
		MaxSensitivity: permission.SensitivityRestricted,
	}, nil
}

func filterOn(filter *pagination.QueryOptions, field string) (domaintypes.FieldFilter, bool) {
	if filter == nil {
		return domaintypes.FieldFilter{}, false
	}
	idx := slices.IndexFunc(filter.FieldFilters, func(f domaintypes.FieldFilter) bool {
		return f.Field == field
	})
	if idx < 0 {
		return domaintypes.FieldFilter{}, false
	}

	return filter.FieldFilters[idx], true
}

// Each ledger list's typed parameters reach its resolver as the filters the
// repository tool applied: the status, the fiscal period, and the
// accounting days in the organization's zone.
func TestLedgerLists_TypedFiltersReachTheOperation(t *testing.T) {
	t.Parallel()

	period := pulid.MustNew("fp_").String()
	entry := pulid.MustNew("je_").String()
	for name, c := range map[string]struct {
		args  map[string]any
		wants map[string]dbtype.Operator
	}{
		toolLedgerEntries: {
			args: map[string]any{
				"status": "Posted", "fiscalPeriodId": period, "referenceType": "Invoice",
				"accountingDateFrom": "2026-03-01", "accountingDateTo": "2026-03-31",
			},
			wants: map[string]dbtype.Operator{
				"status": dbtype.OpEqual, "fiscalPeriodId": dbtype.OpEqual,
				"referenceType":  dbtype.OpContains,
				"accountingDate": dbtype.OpLessThanOrEqual,
			},
		},
		toolLedgerPeriods: {
			args:  map[string]any{"status": []any{"Open"}, "periodType": []any{"Month"}},
			wants: map[string]dbtype.Operator{"status": dbtype.OpIn, "periodType": dbtype.OpIn},
		},
		toolLedgerJournals: {
			args: map[string]any{
				"status": []any{"Approved"}, "requestedFiscalPeriodId": period,
			},
			wants: map[string]dbtype.Operator{
				"status": dbtype.OpIn, "requestedFiscalPeriodId": dbtype.OpEqual,
			},
		},
		toolLedgerReversals: {
			args: map[string]any{
				"status": []any{"Approved"}, "originalJournalEntryId": entry,
			},
			wants: map[string]dbtype.Operator{
				"status": dbtype.OpIn, "originalJournalEntryId": dbtype.OpEqual,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, seen, err := queryLedger(t, name, c.args, permission.SensitivityRestricted)
			require.NoError(t, err)
			filter := seen.last(name)
			require.NotNil(t, filter)
			for field, operator := range c.wants {
				got, ok := filterOn(filter, field)
				require.True(t, ok, "%s filters on %s", name, field)
				if field != "accountingDate" {
					assert.Equal(t, operator, got.Operator, field)
				}
			}
		})
	}

	_, seen, err := queryLedger(t, toolLedgerEntries, map[string]any{
		"accountingDateFrom": "2026-03-01", "accountingDateTo": "2026-03-31",
	}, permission.SensitivityRestricted)
	require.NoError(t, err)
	var from, to int64
	for _, f := range seen.last(toolLedgerEntries).FieldFilters {
		switch f.Operator {
		case dbtype.OpGreaterThanOrEqual:
			from, _ = f.Value.(int64)
		case dbtype.OpLessThanOrEqual:
			to, _ = f.Value.(int64)
		}
	}
	assert.Equal(t, int64(1_772_344_800), from, "the first day opens at the zone's midnight")
	assert.Equal(t, int64(1_775_019_599), to, "the last day closes at the zone's last second")
}

// oldGated are the fields each ledger list's repository tool withheld at the
// agent's default data access (Internal), by the converted tool's names: its
// totals, reasons and notes.
var oldGated = map[string][]string{
	toolLedgerEntries:   {"totalDebit", "totalCredit"},
	toolLedgerPeriods:   {"reopenReason"},
	toolLedgerJournals:  {"totalDebit", "totalCredit", "reason", "rejectionReason", "cancelReason"},
	toolLedgerReversals: {"reasonText", "rejectionReason", "cancelReason"},
}

// shownBefore are what each repository tool showed at Internal, which the
// converted one must still show.
var shownBefore = map[string][]string{
	toolLedgerEntries: {
		"entryNumber", "entryDate", "accountingDate", "entryType", "status", "referenceType",
		"referenceNumber", "description", "fiscalPeriodId",
	},
	toolLedgerPeriods: {
		"fiscalYearId", "name", "periodNumber", "periodType", "status", "startDate",
		"endDate", "reopenedAt",
	},
	toolLedgerJournals: {
		"requestNumber", "status", "description", "accountingDate", "requestedFiscalPeriodId",
		"currencyCode", "isBalanced",
	},
	toolLedgerReversals: {
		"status", "originalJournalEntryId", "requestedAccountingDate",
		"resolvedFiscalPeriodId", "reasonCode",
	},
}

// The converted ledger lists withhold, at the default agent data access,
// every field the repository tools withheld, never show one they hid, and
// name each in withheldByAccess; what the repository tools showed is still
// shown. A filter on a withheld field is refused before the read.
func TestLedgerLists_WithholdAtLeastWhatTheRepositoryToolsWithheld(t *testing.T) {
	t.Parallel()

	for name, fields := range oldGated {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			outcome, _, err := queryLedger(t, name, map[string]any{"limit": 5},
				permission.SensitivityInternal)
			require.NoError(t, err)
			items, _ := outcome["items"].([]any)
			require.Len(t, items, 1)
			row, _ := items[0].(map[string]any)
			for _, field := range fields {
				assert.NotContains(t, row, field, "%s shows %s", name, field)
				assert.Contains(t, outcome["withheldByAccess"], field, name)
			}
			for _, field := range shownBefore[name] {
				assert.Contains(t, row, field, "%s still shows %s", name, field)
			}

			whole, _, err := queryLedger(t, name, map[string]any{"limit": 5},
				permission.SensitivityRestricted)
			require.NoError(t, err)
			wholeItems, _ := whole["items"].([]any)
			require.Len(t, wholeItems, 1)
			wholeRow, _ := wholeItems[0].(map[string]any)
			for _, field := range fields {
				assert.Contains(t, wholeRow, field, "%s shows %s at Restricted", name, field)
			}
		})
	}

	_, seen, err := queryLedger(t, toolLedgerJournals, map[string]any{"totalDebitMin": 100},
		permission.SensitivityInternal)
	require.Error(t, err, "filtering on a withheld total would leak it a row at a time")
	assert.Nil(t, seen.last(toolLedgerJournals), "nothing is read for a refused filter")
}
