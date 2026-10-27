package journalentryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/journalentryportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/journalsourceportsmock"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalsource"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalsourceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewCreatesService(t *testing.T) {
	t.Parallel()

	svc := New(Params{
		Logger:     zap.NewNop(),
		EntryRepo:  journalentryportsmock.NewMockJournalEntryRepository(t),
		SourceRepo: journalsourceportsmock.NewMockJournalSourceRepository(t),
	})

	require.NotNil(t, svc)
}

func TestGetEntryDelegatesToRepository(t *testing.T) {
	t.Parallel()

	entry := &journalentry.JournalEntry{ID: pulid.MustNew("je_")}
	repo := journalentryportsmock.NewMockJournalEntryRepository(t)
	var actualReq journalentryports.GetJournalEntryByIDRequest
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req journalentryports.GetJournalEntryByIDRequest) (*journalentry.JournalEntry, error) {
			actualReq = req
			return entry, nil
		})
	svc := &Service{entryRepo: repo}

	result, err := svc.GetEntry(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		entry.ID,
	)

	require.NoError(t, err)
	assert.Equal(t, entry, result)
	assert.Equal(t, entry.ID, actualReq.ID)
}

func TestGetSourceByObjectDelegatesToRepository(t *testing.T) {
	t.Parallel()

	source := &journalsource.Source{ID: pulid.MustNew("jsrc_")}
	repo := journalsourceportsmock.NewMockJournalSourceRepository(t)
	var actualReq journalsourceports.GetJournalSourceByObjectRequest
	repo.EXPECT().
		GetByObject(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req journalsourceports.GetJournalSourceByObjectRequest) (*journalsource.Source, error) {
			actualReq = req
			return source, nil
		})
	svc := &Service{sourceRepo: repo}

	result, err := svc.GetSourceByObject(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		"Invoice",
		"inv_123",
	)

	require.NoError(t, err)
	assert.Equal(t, source, result)
	assert.Equal(t, "Invoice", actualReq.SourceObjectType)
	assert.Equal(t, "inv_123", actualReq.SourceObjectID)
}

func TestListEntriesDelegatesToRepository(t *testing.T) {
	t.Parallel()

	repo := journalentryportsmock.NewMockJournalEntryRepository(t)
	req := &journalentryports.ListJournalEntriesRequest{
		Filter: &pagination.QueryOptions{Pagination: pagination.Info{Limit: 10}},
	}
	expected := &pagination.ListResult[*journalentry.JournalEntry]{
		Items:      []*journalentry.JournalEntry{{ID: pulid.MustNew("je_")}},
		Total:      1,
		TotalExact: true,
	}
	repo.EXPECT().List(mock.Anything, req).Return(expected, nil).Once()
	svc := &Service{entryRepo: repo}

	result, err := svc.ListEntries(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
