package ledgersync

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errWrite = errors.New("write failed")

type fakeLedger struct {
	posted []*services.AccountingJournalPosted
	err    error
}

func (f *fakeLedger) EnqueueJournal(_ context.Context, posted *services.AccountingJournalPosted) error {
	f.posted = append(f.posted, posted)
	return f.err
}

type fakePosting struct {
	calls int
	err   error
}

//nolint:gocritic // matches the repository contract
func (f *fakePosting) CreatePosting(context.Context, repositories.CreateJournalPostingParams) error {
	f.calls++
	return f.err
}

type fakeReview struct {
	repositories.JournalReviewRepository

	posted []*repositories.PostJournalEntryParams
	err    error
}

func (f *fakeReview) PostEntry(_ context.Context, params *repositories.PostJournalEntryParams) error {
	f.posted = append(f.posted, params)
	return f.err
}

type fakeEntries struct {
	repositories.JournalEntryRepository

	entry *journalentry.JournalEntry
	err   error
	asked []repositories.GetJournalEntryByIDRequest
}

//nolint:gocritic // matches the repository contract
func (f *fakeEntries) GetByID(
	_ context.Context,
	req repositories.GetJournalEntryByIDRequest,
) (*journalentry.JournalEntry, error) {
	f.asked = append(f.asked, req)
	return f.entry, f.err
}

func postingParams(posted bool) repositories.CreateJournalPostingParams {
	return repositories.CreateJournalPostingParams{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		EntryID:        pulid.MustNew("je_"),
		EntryNumber:    "JE-1001",
		EntryType:      journalentry.EntryTypeStandard.String(),
		AccountingDate: 1_790_000_000,
		IsPosted:       posted,
	}
}

func TestCreatePostingQueuesAPostedEntryForTheLedger(t *testing.T) {
	t.Parallel()

	next := &fakePosting{}
	ledger := &fakeLedger{}
	repo := DecoratePosting(next, ledger)
	params := postingParams(true)

	require.NoError(t, repo.CreatePosting(t.Context(), params))

	assert.Equal(t, 1, next.calls)
	require.Len(t, ledger.posted, 1)
	assert.Equal(t, &services.AccountingJournalPosted{
		TenantInfo:     pagination.TenantInfo{OrgID: params.OrganizationID, BuID: params.BusinessUnitID},
		EntryID:        params.EntryID,
		EntryNumber:    "JE-1001",
		EntryType:      journalentry.EntryTypeStandard,
		AccountingDate: params.AccountingDate,
	}, ledger.posted[0])
}

func TestCreatePostingLeavesAnUnpostedOrFailedEntry(t *testing.T) {
	t.Parallel()

	ledger := &fakeLedger{}
	require.NoError(t, DecoratePosting(&fakePosting{}, ledger).CreatePosting(t.Context(), postingParams(false)))
	assert.Empty(t, ledger.posted, "a draft waiting for review is queued when it is posted")

	err := DecoratePosting(&fakePosting{err: errWrite}, ledger).CreatePosting(t.Context(), postingParams(true))
	require.ErrorIs(t, err, errWrite)
	assert.Empty(t, ledger.posted)
}

func TestCreatePostingFailsWhenTheLedgerCannotQueue(t *testing.T) {
	t.Parallel()

	ledger := &fakeLedger{err: errWrite}
	err := DecoratePosting(&fakePosting{}, ledger).CreatePosting(t.Context(), postingParams(true))
	require.ErrorIs(t, err, errWrite, "the posting and its record commit together or not at all")
}

func TestPostEntryQueuesTheReviewedEntry(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	entry := &journalentry.JournalEntry{
		ID:             pulid.MustNew("je_"),
		EntryNumber:    "JE-2002",
		EntryType:      journalentry.EntryTypeAdjusting,
		AccountingDate: 1_790_086_400,
	}
	review := &fakeReview{}
	entries := &fakeEntries{entry: entry}
	ledger := &fakeLedger{}
	repo := DecorateReview(review, entries, ledger)

	require.NoError(t, repo.PostEntry(t.Context(), &repositories.PostJournalEntryParams{
		TenantInfo: tenantInfo,
		EntryID:    entry.ID,
	}))

	require.Len(t, review.posted, 1)
	require.Len(t, entries.asked, 1)
	assert.Equal(t, entry.ID, entries.asked[0].ID)
	assert.Equal(t, tenantInfo, entries.asked[0].TenantInfo)
	require.Len(t, ledger.posted, 1)
	assert.Equal(t, &services.AccountingJournalPosted{
		TenantInfo:     tenantInfo,
		EntryID:        entry.ID,
		EntryNumber:    "JE-2002",
		EntryType:      journalentry.EntryTypeAdjusting,
		AccountingDate: entry.AccountingDate,
	}, ledger.posted[0])
}

func TestPostEntryQueuesNothingWhenPostingFails(t *testing.T) {
	t.Parallel()

	ledger := &fakeLedger{}
	entries := &fakeEntries{}
	err := DecorateReview(&fakeReview{err: errWrite}, entries, ledger).PostEntry(
		t.Context(),
		&repositories.PostJournalEntryParams{EntryID: pulid.MustNew("je_")},
	)
	require.ErrorIs(t, err, errWrite)
	assert.Empty(t, entries.asked)
	assert.Empty(t, ledger.posted)
}

func TestAReversalCarriesTheEntryItReverses(t *testing.T) {
	t.Parallel()

	ledger := &fakeLedger{}
	params := postingParams(true)
	params.EntryType = journalentry.EntryTypeReversal.String()
	params.IsReversal = true
	params.ReversalOfID = pulid.MustNew("je_")
	require.NoError(t, DecoratePosting(&fakePosting{}, ledger).CreatePosting(t.Context(), params))

	require.Len(t, ledger.posted, 1)
	assert.Equal(t, params.ReversalOfID, ledger.posted[0].ReversalOfID)

	stale := postingParams(true)
	stale.ReversalOfID = pulid.MustNew("je_")
	require.NoError(t, DecoratePosting(&fakePosting{}, ledger).CreatePosting(t.Context(), stale))
	assert.True(t, ledger.posted[1].ReversalOfID.IsNil(), "only a reversal names what it reverses")

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	reviewed := &journalentry.JournalEntry{
		ID:           pulid.MustNew("je_"),
		EntryNumber:  "JE-3003",
		EntryType:    journalentry.EntryTypeReversal,
		IsReversal:   true,
		ReversalOfID: pulid.MustNew("je_"),
	}
	repo := DecorateReview(&fakeReview{}, &fakeEntries{entry: reviewed}, ledger)
	require.NoError(t, repo.PostEntry(t.Context(), &repositories.PostJournalEntryParams{
		TenantInfo: tenantInfo,
		EntryID:    reviewed.ID,
	}))
	require.Len(t, ledger.posted, 3)
	assert.Equal(t, reviewed.ReversalOfID, ledger.posted[2].ReversalOfID)
}
