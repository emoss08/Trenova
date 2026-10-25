//go:build integration

package journalentryrepository

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/accountingtest"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/testutil/fiscaltest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type journalEntryFixture struct {
	db      *accountingtest.Database
	repo    journalentryports.JournalEntryRepository
	period  pulid.ID
	invoice *journalentry.JournalEntry
	manual  *journalentry.JournalEntry
	entryAt int64
}

func newJournalEntryFixture(t *testing.T) *journalEntryFixture {
	t.Helper()

	db := accountingtest.NewDatabase(t)
	entryAt := time.Date(2021, time.March, 15, 12, 0, 0, 0, time.UTC).Unix()
	fy, fp := fiscaltest.OpenPeriod(t, t.Context(), fiscaltest.OpenPeriodParams{
		DB:             db.Admin,
		OrganizationID: db.A.Info.OrgID,
		BusinessUnitID: db.A.Info.BuID,
		At:             entryAt,
	})

	ctx := db.A.Ctx
	batch := &journalentry.JournalBatch{
		BatchNumber:    "JB-LIST-1",
		BatchType:      "System",
		Status:         "Posted",
		Description:    "Batch",
		AccountingDate: entryAt,
		FiscalYearID:   fy.ID,
		FiscalPeriodID: fp.ID,
		CreatedByID:    db.A.Info.UserID,
	}
	require.NoError(t, journalentrydb.JournalBatches.Create(ctx, db.Admin, batch))

	entry := func(number string, at int64, status journalentry.Status, refType string, amount int64) *journalentry.JournalEntry {
		e := &journalentry.JournalEntry{
			BatchID:        batch.ID,
			FiscalYearID:   fy.ID,
			FiscalPeriodID: fp.ID,
			EntryNumber:    number,
			EntryDate:      at,
			EntryType:      journalentry.EntryTypeStandard,
			AccountingDate: at,
			Status:         status,
			ReferenceType:  refType,
			ReferenceID:    pulid.MustNew("ref_").String(),
			Description:    refType + " entry",
			TotalDebit:     amount,
			TotalCredit:    amount,
			CreatedByID:    db.A.Info.UserID,
		}
		require.NoError(t, journalentrydb.JournalEntries.Create(ctx, db.Admin, e))
		return e
	}

	return &journalEntryFixture{
		db:      db,
		repo:    New(Params{DB: db.Conn, Logger: zap.NewNop()}),
		period:  fp.ID,
		invoice: entry("JE-100", entryAt, journalentry.StatusPosted, "InvoicePosted", 100),
		manual:  entry("JE-200", entryAt+10, journalentry.StatusPending, "ManualJournalPosted", 200),
		entryAt: entryAt,
	}
}

func TestListJournalEntries(t *testing.T) {
	f := newJournalEntryFixture(t)

	result, err := f.repo.List(f.db.A.Ctx, &journalentryports.ListJournalEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.db.A.Info,
			Pagination: pagination.Info{Limit: 10},
		},
		FiscalPeriodID: f.period,
		Status:         "Posted",
		ReferenceType:  "InvoicePosted",
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, f.invoice.ID, result.Items[0].ID)
	assert.Equal(t, 1, result.Total)
}

func TestListJournalEntriesSearchesAndOrdersNewestFirst(t *testing.T) {
	f := newJournalEntryFixture(t)

	result, err := f.repo.List(f.db.A.Ctx, &journalentryports.ListJournalEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.db.A.Info,
			Query:      "je-",
			Pagination: pagination.Info{Limit: 10},
		},
		AccountingDateStart: f.entryAt,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, f.manual.ID, result.Items[0].ID)
	assert.Equal(t, f.invoice.ID, result.Items[1].ID)
	assert.Equal(t, 2, result.Total)

	other, err := f.repo.List(f.db.B.Ctx, &journalentryports.ListJournalEntriesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: f.db.B.Info,
			Pagination: pagination.Info{Limit: 10},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, other.Items)
	assert.Equal(t, 0, other.Total)
}

func (f *journalEntryFixture) connection(
	t *testing.T,
	limit int,
	after string,
) *pagination.CursorListResult[*journalentry.JournalEntry] {
	t.Helper()

	cursor, err := pagination.NewCursorInfo(limit, after)
	require.NoError(t, err)
	cursor.IncludeTotalCount = true
	result, err := f.repo.ListConnection(f.db.A.Ctx, &journalentryports.ListJournalEntryConnectionRequest{
		Filter: &pagination.QueryOptions{TenantInfo: f.db.A.Info},
		Cursor: cursor,
	})
	require.NoError(t, err)

	return result
}

func TestListConnectionPagesThroughEveryEntry(t *testing.T) {
	f := newJournalEntryFixture(t)

	first := f.connection(t, 1, "")
	require.Len(t, first.Items, 1)
	assert.Equal(t, f.manual.ID, first.Items[0].ID)
	require.NotNil(t, first.TotalCount)
	assert.Equal(t, 2, *first.TotalCount)
	require.True(t, first.HasNextPage)

	next, err := first.NextCursor()
	require.NoError(t, err)
	second := f.connection(t, 1, next)
	require.Len(t, second.Items, 1)
	assert.Equal(t, f.invoice.ID, second.Items[0].ID)
	assert.False(t, second.HasNextPage)
}

func TestMarkReversedStampsTheOriginalOnce(t *testing.T) {
	f := newJournalEntryFixture(t)
	reverse := journalentryports.MarkJournalEntryReversedRequest{
		OriginalEntryID: f.invoice.ID,
		ReversalEntryID: f.manual.ID,
		ReversalDate:    f.entryAt + 100,
		ReversalReason:  "Posted twice",
		UpdatedByID:     f.db.A.Info.UserID,
	}
	require.NoError(t, f.repo.MarkReversed(f.db.A.Ctx, reverse))

	stored, err := f.repo.GetByID(f.db.A.Ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         f.invoice.ID,
		TenantInfo: f.db.A.Info,
	})
	require.NoError(t, err)
	assert.Equal(t, journalentry.StatusReversed, stored.Status)
	assert.Equal(t, f.manual.ID, stored.ReversedByID)
	require.NotNil(t, stored.ReversalDate)
	assert.Equal(t, f.entryAt+100, *stored.ReversalDate)
	assert.Equal(t, f.invoice.Version+1, stored.Version)

	reverse.ReversalReason = "Second attempt"
	require.NoError(t, f.repo.MarkReversed(f.db.A.Ctx, reverse))
	again, err := f.repo.GetByID(f.db.A.Ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         f.invoice.ID,
		TenantInfo: f.db.A.Info,
	})
	require.NoError(t, err)
	assert.Equal(t, "Posted twice", again.ReversalReason)
	assert.Equal(t, stored.Version, again.Version)
}

func TestGetByIDStaysInsideTheTenant(t *testing.T) {
	f := newJournalEntryFixture(t)

	_, err := f.repo.GetByID(f.db.B.Ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         f.invoice.ID,
		TenantInfo: f.db.B.Info,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestReversibleSelectOptionsListOnlyPostedEntriesNotYetReversedNewestFirst(t *testing.T) {
	f := newJournalEntryFixture(t)
	ctx := f.db.A.Ctx
	je := journalentrydb.JournalEntries

	entry := func(number string, status journalentry.Status, at int64, posted bool) *journalentry.JournalEntry {
		e := &journalentry.JournalEntry{
			BatchID:        f.invoice.BatchID,
			FiscalYearID:   f.invoice.FiscalYearID,
			FiscalPeriodID: f.period,
			EntryNumber:    number,
			EntryDate:      at,
			EntryType:      journalentry.EntryTypeStandard,
			AccountingDate: at,
			Status:         status,
			IsPosted:       posted,
			ReferenceType:  "ManualJournalPosted",
			ReferenceID:    pulid.MustNew("mjr_").String(),
			Description:    "Manual journal",
			TotalDebit:     100,
			TotalCredit:    100,
			CreatedByID:    f.db.A.Info.UserID,
		}
		if posted {
			e.PostedAt = &at
			e.PostedByID = f.db.A.Info.UserID
		}
		require.NoError(t, je.Create(ctx, f.db.Admin, e))
		return e
	}

	day := int64(86400)
	entry("JE-REV-510", journalentry.StatusPosted, f.entryAt, true)
	entry("JE-REV-520", journalentry.StatusPosted, f.entryAt+day, true)
	entry("JE-REV-530", journalentry.StatusPending, f.entryAt+2*day, false)
	reversed := entry("JE-REV-540", journalentry.StatusReversed, f.entryAt+3*day, true)
	reversalAt := f.entryAt + 4*day
	reversal := &journalentry.JournalEntry{
		BatchID:        f.invoice.BatchID,
		FiscalYearID:   f.invoice.FiscalYearID,
		FiscalPeriodID: f.period,
		EntryNumber:    "JE-REV-550",
		EntryDate:      f.entryAt + 4*day,
		EntryType:      journalentry.EntryTypeStandard,
		AccountingDate: f.entryAt + 4*day,
		Status:         journalentry.StatusPosted,
		IsPosted:       true,
		PostedAt:       &reversalAt,
		PostedByID:     f.db.A.Info.UserID,
		IsReversal:     true,
		ReversalOfID:   reversed.ID,
		ReferenceType:  "ManualJournalPosted",
		ReferenceID:    pulid.MustNew("mjr_").String(),
		Description:    "Manual journal",
		TotalDebit:     100,
		TotalCredit:    100,
		CreatedByID:    f.db.A.Info.UserID,
	}
	require.NoError(t, je.Create(ctx, f.db.Admin, reversal))
	_, err := je.Where(je.ID.Eq(reversed.ID)).Update(ctx, f.db.Admin, je.ReversedByID.To(reversal.ID))
	require.NoError(t, err)

	numbers := func(query string, reversibleOnly bool) []string {
		result, listErr := f.repo.SelectOptions(ctx, &journalentryports.JournalEntrySelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: f.db.A.Info,
				Pagination: pagination.Info{Limit: 10},
				Query:      query,
			},
			ReversibleOnly: reversibleOnly,
		})
		require.NoError(t, listErr)
		got := make([]string, 0, len(result.Items))
		for _, item := range result.Items {
			got = append(got, item.EntryNumber)
		}
		return got
	}

	assert.Equal(t, []string{"JE-REV-520", "JE-REV-510"}, numbers("je-rev-5", true))
	assert.Equal(t, []string{"JE-REV-510"}, numbers("510", true))
	assert.Equal(t,
		[]string{"JE-REV-550", "JE-REV-540", "JE-REV-530", "JE-REV-520", "JE-REV-510"},
		numbers("je-rev-5", false),
	)

	got, err := f.repo.GetByIDs(ctx, journalentryports.GetJournalEntriesByIDsRequest{
		TenantInfo: f.db.A.Info,
		IDs:        []pulid.ID{reversed.ID, reversal.ID},
	})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	got, err = f.repo.GetByIDs(f.db.B.Ctx, journalentryports.GetJournalEntriesByIDsRequest{
		TenantInfo: f.db.B.Info,
		IDs:        []pulid.ID{reversed.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, got, "another tenant reads none of the entries")
}
