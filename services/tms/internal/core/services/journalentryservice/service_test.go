package journalentryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
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
		EntryRepo:  mocks.NewMockJournalEntryRepository(t),
		SourceRepo: mocks.NewMockJournalSourceRepository(t),
	})

	require.NotNil(t, svc)
}

func TestGetEntryDelegatesToRepository(t *testing.T) {
	t.Parallel()

	entry := &journalentry.Entry{ID: pulid.MustNew("je_")}
	repo := mocks.NewMockJournalEntryRepository(t)
	var actualReq repositories.GetJournalEntryByIDRequest
	repo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetJournalEntryByIDRequest) (*journalentry.Entry, error) {
			actualReq = req
			return entry, nil
		})
	svc := &Service{entryRepo: repo}

	result, err := svc.GetEntry(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}, entry.ID)

	require.NoError(t, err)
	assert.Equal(t, entry, result)
	assert.Equal(t, entry.ID, actualReq.ID)
}

func TestGetSourceByObjectDelegatesToRepository(t *testing.T) {
	t.Parallel()

	source := &journalsource.Source{ID: pulid.MustNew("jsrc_")}
	repo := mocks.NewMockJournalSourceRepository(t)
	var actualReq repositories.GetJournalSourceByObjectRequest
	repo.EXPECT().
		GetByObject(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repositories.GetJournalSourceByObjectRequest) (*journalsource.Source, error) {
			actualReq = req
			return source, nil
		})
	svc := &Service{sourceRepo: repo}

	result, err := svc.GetSourceByObject(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}, "Invoice", "inv_123")

	require.NoError(t, err)
	assert.Equal(t, source, result)
	assert.Equal(t, "Invoice", actualReq.SourceObjectType)
	assert.Equal(t, "inv_123", actualReq.SourceObjectID)
}
