package journalentryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewCreatesService(t *testing.T) {
	t.Parallel()

	svc := New(Params{Logger: zap.NewNop(), EntryRepo: &fakeEntryRepo{}, SourceRepo: &fakeSourceRepo{}})

	require.NotNil(t, svc)
}

func TestGetEntryDelegatesToRepository(t *testing.T) {
	t.Parallel()

	entry := &journalentry.Entry{ID: pulid.MustNew("je_")}
	repo := &fakeEntryRepo{entry: entry}
	svc := &Service{entryRepo: repo}

	result, err := svc.GetEntry(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}, entry.ID)

	require.NoError(t, err)
	assert.Equal(t, entry, result)
	assert.Equal(t, entry.ID, repo.lastEntryReq.ID)
}

func TestGetSourceByObjectDelegatesToRepository(t *testing.T) {
	t.Parallel()

	source := &journalsource.Source{ID: pulid.MustNew("jsrc_")}
	repo := &fakeSourceRepo{source: source}
	svc := &Service{sourceRepo: repo}

	result, err := svc.GetSourceByObject(t.Context(), pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}, "Invoice", "inv_123")

	require.NoError(t, err)
	assert.Equal(t, source, result)
	assert.Equal(t, "Invoice", repo.lastSourceReq.SourceObjectType)
	assert.Equal(t, "inv_123", repo.lastSourceReq.SourceObjectID)
}

type fakeEntryRepo struct {
	entry        *journalentry.Entry
	lastEntryReq repositories.GetJournalEntryByIDRequest
}

func (f *fakeEntryRepo) GetByID(_ context.Context, req repositories.GetJournalEntryByIDRequest) (*journalentry.Entry, error) {
	f.lastEntryReq = req
	return f.entry, nil
}

func (f *fakeEntryRepo) MarkReversed(context.Context, repositories.MarkJournalEntryReversedRequest) error {
	return nil
}

type fakeSourceRepo struct {
	source        *journalsource.Source
	lastSourceReq repositories.GetJournalSourceByObjectRequest
}

func (f *fakeSourceRepo) GetByObject(_ context.Context, req repositories.GetJournalSourceByObjectRequest) (*journalsource.Source, error) {
	f.lastSourceReq = req
	return f.source, nil
}
