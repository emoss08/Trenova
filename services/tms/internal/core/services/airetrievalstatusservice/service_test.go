package airetrievalstatusservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/memtable"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testNow      = int64(1_790_000_000)
	activeKey    = "api.voyageai.com/voyage-3.5@1024"
	pendingKey   = "api.openai.com/text-embedding-3-small@1536"
	someFailure  = "provider refused the request"
	budgetString = "10.00"
)

func int64p(v int64) *int64 { return &v }

func entryCount(
	sourceType airetrieval.SourceType,
	status airetrieval.IndexStatus,
	count int,
) repositories.IndexEntryCount {
	return repositories.IndexEntryCount{SourceType: sourceType, Status: status, Count: count}
}

func stamped(
	count repositories.IndexEntryCount,
	lastIndexedAt, lastAttemptAt *int64,
) repositories.IndexEntryCount {
	count.LastIndexedAt = lastIndexedAt
	count.LastAttemptAt = lastAttemptAt

	return count
}

type fakeRepo struct {
	repositories.AIRetrievalRepository

	mu            sync.Mutex
	settings      *airetrieval.Settings
	counts        map[string][]repositories.IndexEntryCount
	countedKeys   []string
	averages      map[airetrieval.SourceType]repositories.IndexChunkAverage
	errored       []*airetrieval.IndexEntry
	erroredReq    *repositories.ListErroredIndexEntriesRequest
	updates       []*airetrieval.Settings
	mismatchFirst int
	settingsErr   error
}

func (r *fakeRepo) GetSettings(
	_ context.Context,
	tenant pagination.TenantInfo,
) (*airetrieval.Settings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.settingsErr != nil {
		return nil, r.settingsErr
	}
	if r.settings == nil {
		return airetrieval.DefaultSettings(tenant.OrgID, tenant.BuID), nil
	}
	copied := *r.settings

	return &copied, nil
}

func (r *fakeRepo) UpdateSettings(
	_ context.Context,
	entity *airetrieval.Settings,
) (*airetrieval.Settings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mismatchFirst > 0 {
		r.mismatchFirst--
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Version mismatch",
		)
	}

	saved := *entity
	saved.Version++
	r.settings = &saved
	r.updates = append(r.updates, &saved)
	copied := saved

	return &copied, nil
}

func (r *fakeRepo) CountIndexEntries(
	_ context.Context,
	req repositories.CountIndexEntriesRequest,
) ([]repositories.IndexEntryCount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.countedKeys = append(r.countedKeys, req.ModelKey)

	return r.counts[req.ModelKey], nil
}

func (r *fakeRepo) AverageIndexChunks(
	_ context.Context,
	req *repositories.AverageIndexChunksRequest,
) (repositories.IndexChunkAverage, error) {
	return r.averages[req.SourceType], nil
}

func (r *fakeRepo) ListErroredIndexEntries(
	_ context.Context,
	req *repositories.ListErroredIndexEntriesRequest,
) ([]*airetrieval.IndexEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.erroredReq = req

	return r.errored, nil
}

type fakeSources struct {
	repositories.RetrievalSourceRepository

	totals map[airetrieval.SourceType]int
	chars  map[airetrieval.SourceType]float64
}

func (s *fakeSources) CountSources(
	_ context.Context,
	req repositories.CountRetrievalSourcesRequest,
) (int, error) {
	return s.totals[req.SourceType], nil
}

func (s *fakeSources) AverageSourceChars(
	_ context.Context,
	req repositories.AverageRetrievalSourceCharsRequest,
) (repositories.RetrievalSourceChars, error) {
	return repositories.RetrievalSourceChars{Sampled: 10, AverageChars: s.chars[req.SourceType]}, nil
}

type fakeUsage struct {
	repositories.AIUsageRepository

	mu       sync.Mutex
	costs    map[aiusage.Surface]*repositories.AIUsageCost
	requests []repositories.AIUsageSurfaceCostRequest
}

func (u *fakeUsage) SurfaceCost(
	_ context.Context,
	req repositories.AIUsageSurfaceCostRequest,
) (*repositories.AIUsageCost, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.requests = append(u.requests, req)
	if cost, ok := u.costs[req.Surface]; ok {
		return cost, nil
	}

	return &repositories.AIUsageCost{}, nil
}

type fakeProviders struct {
	repositories.AIProviderRepository

	providers []*aiprovider.Provider
}

func (p *fakeProviders) ListForTask(
	_ context.Context,
	_ repositories.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	return p.providers, nil
}

type fakeVectorizer struct {
	serviceports.QueryVectorizer

	availability airetrieval.Availability
}

func (v *fakeVectorizer) Availability(
	_ context.Context,
	_ pagination.TenantInfo,
) (airetrieval.Availability, error) {
	return v.availability, nil
}

type fakeIndexer struct {
	serviceports.RetrievalIndexer

	reindexed []airetrieval.SourceType
}

func (i *fakeIndexer) Reindex(
	_ context.Context,
	_ pagination.TenantInfo,
	sourceType airetrieval.SourceType,
) error {
	i.reindexed = append(i.reindexed, sourceType)
	return nil
}

type fakePipeline struct {
	serviceports.RetrievalIndexPipeline

	woken int
}

func (p *fakePipeline) Wake(_ context.Context, _ pagination.TenantInfo) { p.woken++ }

type fakeAudit struct {
	serviceports.AuditService

	logged []*serviceports.LogActionParams
}

func (a *fakeAudit) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	a.logged = append(a.logged, params)
	return nil
}

type fakeEmbeddings struct {
	serviceports.EmbeddingService

	key string
	err error
}

func (e *fakeEmbeddings) ConfiguredModelKey(
	_ context.Context,
	_ pagination.TenantInfo,
) (string, error) {
	return e.key, e.err
}

type fixture struct {
	tenant     pagination.TenantInfo
	repo       *fakeRepo
	sources    *fakeSources
	usage      *fakeUsage
	providers  *fakeProviders
	vectorizer *fakeVectorizer
	indexer    *fakeIndexer
	pipeline   *fakePipeline
	audit      *fakeAudit
	embeddings *fakeEmbeddings
	svc        *Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		tenant:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		repo:      &fakeRepo{counts: map[string][]repositories.IndexEntryCount{}},
		sources:   &fakeSources{totals: map[airetrieval.SourceType]int{}},
		usage:     &fakeUsage{costs: map[aiusage.Surface]*repositories.AIUsageCost{}},
		providers: &fakeProviders{},
		vectorizer: &fakeVectorizer{
			availability: airetrieval.Availability{Available: true, ExtensionInstalled: true},
		},
		indexer:    &fakeIndexer{},
		pipeline:   &fakePipeline{},
		audit:      &fakeAudit{},
		embeddings: &fakeEmbeddings{key: activeKey},
	}
	f.svc = New(Params{
		Logger:     zap.NewNop(),
		Repo:       f.repo,
		Sources:    f.sources,
		Usage:      f.usage,
		Providers:  f.providers,
		Vectorizer: f.vectorizer,
		Indexer:    f.indexer,
		Pipeline:   f.pipeline,
		Audit:      f.audit,
		Embeddings: f.embeddings,
	})
	f.svc.now = func() int64 { return testNow }

	return f
}

func (f *fixture) withSettings(mutate func(*airetrieval.Settings)) {
	settings := airetrieval.DefaultSettings(f.tenant.OrgID, f.tenant.BuID)
	settings.ID = pulid.MustNew("airs_")
	settings.ActiveModelKey = activeKey
	settings.Dimensions = 1024
	settings.MonthlyIndexingBudgetUSD = decimal.RequireFromString(budgetString)
	if mutate != nil {
		mutate(settings)
	}
	f.repo.settings = settings
}

func sourceOf(
	status *serviceports.AIRetrievalStatus,
	sourceType airetrieval.SourceType,
) *serviceports.AIRetrievalSourceStatus {
	for _, source := range status.Sources {
		if source.SourceType == sourceType {
			return source
		}
	}

	return nil
}

func TestStatus_CountsEachSourceUnderTheActiveModel(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) { s.InboundMessagesEnabled = false })
	f.sources.totals = map[airetrieval.SourceType]int{
		airetrieval.SourceTypeMemory:         40,
		airetrieval.SourceTypeDocument:       120,
		airetrieval.SourceTypeInboundMessage: 7,
	}
	f.repo.counts[activeKey] = []repositories.IndexEntryCount{
		stamped(
			entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusIndexed, 30),
			int64p(1_000),
			nil,
		),
		entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusPending, 8),
		stamped(
			entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusSkipped, 2),
			int64p(1_500),
			nil,
		),
		stamped(
			entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusIndexed, 100),
			int64p(2_000),
			int64p(2_000),
		),
		stamped(
			entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusFailed, 5),
			nil,
			int64p(2_500),
		),
	}

	status, err := f.svc.Status(t.Context(), f.tenant)
	require.NoError(t, err)

	require.Len(t, status.Sources, 3)
	assert.Equal(t, []airetrieval.SourceType{
		airetrieval.SourceTypeMemory,
		airetrieval.SourceTypeDocument,
		airetrieval.SourceTypeInboundMessage,
	}, []airetrieval.SourceType{
		status.Sources[0].SourceType,
		status.Sources[1].SourceType,
		status.Sources[2].SourceType,
	})

	memory := sourceOf(status, airetrieval.SourceTypeMemory)
	assert.True(t, memory.Enabled)
	assert.Equal(t, 40, memory.Total)
	assert.Equal(t, 30, memory.Indexed)
	assert.Equal(t, 8, memory.Pending)
	assert.Equal(t, 2, memory.Skipped)
	assert.Equal(t, 0, memory.Failed)
	require.NotNil(t, memory.LastIndexedAt)
	assert.Equal(t, int64(1_500), *memory.LastIndexedAt)

	document := sourceOf(status, airetrieval.SourceTypeDocument)
	assert.Equal(t, 100, document.Indexed)
	assert.Equal(t, 5, document.Failed)
	require.NotNil(t, document.LastAttemptAt)
	assert.Equal(t, int64(2_500), *document.LastAttemptAt)

	inbound := sourceOf(status, airetrieval.SourceTypeInboundMessage)
	assert.False(t, inbound.Enabled)
	assert.Equal(t, 7, inbound.Total)
	assert.Nil(t, inbound.LastIndexedAt)

	require.NotNil(t, status.LastIndexedAt)
	assert.Equal(t, int64(2_000), *status.LastIndexedAt)
	assert.Equal(t, []string{activeKey}, f.repo.countedKeys)
	assert.Nil(t, status.ModelChange)
}

func TestStatus_WithoutAnActiveModelReadsNoEntries(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.embeddings.err = serviceports.ErrNoProviderConfigured
	f.vectorizer.availability = airetrieval.Availability{
		Reason:             airetrieval.UnavailableReasonNoProvider,
		ExtensionInstalled: true,
	}

	status, err := f.svc.Status(t.Context(), f.tenant)
	require.NoError(t, err)

	assert.Empty(t, f.repo.countedKeys)
	for _, source := range status.Sources {
		assert.Zero(t, source.Indexed+source.Pending+source.Failed+source.Skipped)
	}
	assert.Nil(t, status.Settings.ActiveModelKey)
	assert.Nil(t, status.Settings.Dimensions)
	assert.Nil(t, status.ConfiguredModelKey)
	assert.False(t, status.ConfiguredModelDiffers)
	assert.Nil(t, status.LastIndexedAt)
}

func TestStatus_ReportsEveryUnavailableReason(t *testing.T) {
	t.Parallel()

	for _, reason := range airetrieval.AllUnavailableReasons() {
		t.Run(reason.String(), func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.vectorizer.availability = airetrieval.Availability{
				Reason:             reason,
				ExtensionInstalled: reason != airetrieval.UnavailableReasonExtensionMissing,
				ExtensionVersion:   "0.7.4",
			}

			status, err := f.svc.Status(t.Context(), f.tenant)
			require.NoError(t, err)

			assert.False(t, status.Availability.Available)
			require.NotNil(t, status.Availability.Reason)
			assert.Equal(t, reason, *status.Availability.Reason)
			assert.Equal(
				t,
				reason != airetrieval.UnavailableReasonExtensionMissing,
				status.Availability.ExtensionInstalled,
			)
			require.NotNil(t, status.Availability.ExtensionVersion)
			assert.Equal(t, "0.7.4", *status.Availability.ExtensionVersion)
		})
	}

	t.Run("available", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		status, err := f.svc.Status(t.Context(), f.tenant)
		require.NoError(t, err)

		assert.True(t, status.Availability.Available)
		assert.Nil(t, status.Availability.Reason)
		assert.Nil(t, status.Availability.ExtensionVersion)
	})
}

func TestStatus_SumsThisMonthsIndexingAndRetrievalCost(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	f.usage.costs[aiusage.SurfaceIndexing] = &repositories.AIUsageCost{
		CostUSD:       decimal.RequireFromString("1.23456"),
		Calls:         12,
		UnpricedCalls: 2,
	}
	f.usage.costs[aiusage.SurfaceRetrieval] = &repositories.AIUsageCost{
		CostUSD: decimal.RequireFromString("0.004"),
		Calls:   40,
	}

	status, err := f.svc.Status(t.Context(), f.tenant)
	require.NoError(t, err)

	assert.Equal(t, "1.23", status.IndexingCostMonthUSD)
	assert.Equal(t, 2, status.IndexingUnpricedCalls)
	assert.Equal(t, "0.00", status.RetrievalCostMonthUSD)
	assert.Equal(t, 0, status.RetrievalUnpricedCalls)
	assert.Equal(t, timeutils.MonthStartUTC(testNow), status.MonthStartedAt)
	assert.Equal(t, budgetString, status.Settings.MonthlyIndexingBudgetUSD)

	require.Len(t, f.usage.requests, 2)
	for _, req := range f.usage.requests {
		assert.Equal(t, timeutils.MonthStartUTC(testNow), req.Since)
		assert.Equal(t, f.tenant, req.TenantInfo)
	}
	assert.ElementsMatch(t,
		[]aiusage.Surface{aiusage.SurfaceIndexing, aiusage.SurfaceRetrieval},
		[]aiusage.Surface{f.usage.requests[0].Surface, f.usage.requests[1].Surface},
	)
}

func TestStatus_ReportsModelChangeProgressOverEnabledSources(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) {
		s.PendingModelKey = pendingKey
		s.PendingDimensions = 1536
		s.InboundMessagesEnabled = false
	})
	f.embeddings.key = pendingKey
	f.sources.totals = map[airetrieval.SourceType]int{
		airetrieval.SourceTypeMemory:         10,
		airetrieval.SourceTypeDocument:       30,
		airetrieval.SourceTypeInboundMessage: 500,
	}
	f.repo.counts[pendingKey] = []repositories.IndexEntryCount{
		entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusIndexed, 9),
		entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusSkipped, 1),
		entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusIndexed, 12),
		entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusPending, 15),
		entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusFailed, 3),
		entryCount(airetrieval.SourceTypeInboundMessage, airetrieval.IndexStatusIndexed, 400),
	}

	status, err := f.svc.Status(t.Context(), f.tenant)
	require.NoError(t, err)

	require.NotNil(t, status.ModelChange)
	assert.Equal(t, &serviceports.AIRetrievalModelChange{
		FromModelKey: activeKey,
		ToModelKey:   pendingKey,
		Dimensions:   1536,
		Total:        40,
		Indexed:      22,
		Pending:      15,
		Failed:       3,
	}, status.ModelChange)
	assert.ElementsMatch(t, []string{activeKey, pendingKey}, f.repo.countedKeys)
	require.NotNil(t, status.ConfiguredModelKey)
	assert.Equal(t, pendingKey, *status.ConfiguredModelKey)
	assert.True(t, status.ConfiguredModelDiffers)
	require.NotNil(t, status.Settings.PendingModelKey)
	assert.Equal(t, pendingKey, *status.Settings.PendingModelKey)
}

func TestModelChangeProgress_NeverReportsMoreDoneThanTotal(t *testing.T) {
	t.Parallel()

	settings := airetrieval.DefaultSettings(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	settings.ActiveModelKey = activeKey
	settings.Dimensions = 1024
	settings.PendingModelKey = pendingKey
	settings.PendingDimensions = 1536

	change := ModelChangeProgress(
		settings,
		map[airetrieval.SourceType]int{airetrieval.SourceTypeMemory: 2},
		[]repositories.IndexEntryCount{
			entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusIndexed, 5),
		},
	)

	require.NotNil(t, change)
	assert.Equal(t, 5, change.Total)
	assert.Equal(t, 5, change.Indexed)
	unchanged := airetrieval.DefaultSettings(settings.OrganizationID, settings.BusinessUnitID)
	assert.Nil(t, ModelChangeProgress(unchanged, nil, nil))
}

func TestStatus_ConfiguredModelErrors(t *testing.T) {
	t.Parallel()

	t.Run("a business refusal reads as no configured model", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.withSettings(nil)
		f.embeddings.err = errortypes.NewBusinessError("AI is turned off")

		status, err := f.svc.Status(t.Context(), f.tenant)
		require.NoError(t, err)
		assert.Nil(t, status.ConfiguredModelKey)
		assert.False(t, status.ConfiguredModelDiffers)
	})

	t.Run("an unexpected failure is returned", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.embeddings.err = errors.New("database gone")

		_, err := f.svc.Status(t.Context(), f.tenant)
		require.Error(t, err)
	})

	t.Run("the configured model matching the active one does not differ", func(t *testing.T) {
		t.Parallel()

		f := newFixture(t)
		f.withSettings(nil)

		status, err := f.svc.Status(t.Context(), f.tenant)
		require.NoError(t, err)
		assert.False(t, status.ConfiguredModelDiffers)
	})
}

func TestStatus_RequiresATenant(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.Status(t.Context(), pagination.TenantInfo{})
	require.Error(t, err)
}

func boolp(v bool) *bool { return &v }

func TestUpdateSettings_ChangesOnlyTheGivenFields(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	budget := decimal.RequireFromString("42.5")

	status, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:               f.tenant,
		DocumentsEnabled:         boolp(false),
		MonthlyIndexingBudgetUSD: &budget,
		Actor:                    &serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
	})
	require.NoError(t, err)

	require.Len(t, f.repo.updates, 1)
	saved := f.repo.updates[0]
	assert.False(t, saved.DocumentsEnabled)
	assert.True(t, saved.MemoryEnabled)
	assert.True(t, saved.InboundMessagesEnabled)
	assert.True(t, budget.Equal(saved.MonthlyIndexingBudgetUSD))
	assert.Equal(t, activeKey, saved.ActiveModelKey, "the model is the indexer's, never the patch's")
	assert.Equal(t, "42.50", status.Settings.MonthlyIndexingBudgetUSD)
	assert.False(t, status.Settings.DocumentsEnabled)

	require.Len(t, f.audit.logged, 1)
	assert.Equal(t, permission.ResourceAIProvider, f.audit.logged[0].Resource)
	assert.Equal(t, permission.OpUpdate, f.audit.logged[0].Operation)
	assert.Equal(t, 1, f.pipeline.woken)
}

func TestUpdateSettings_NothingChangedWritesNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)

	_, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:    f.tenant,
		MemoryEnabled: boolp(true),
	})
	require.NoError(t, err)

	assert.Empty(t, f.repo.updates)
	assert.Empty(t, f.audit.logged)
	assert.Zero(t, f.pipeline.woken)
}

func TestUpdateSettings_RetriesWhenTheIndexerWroteFirst(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	f.repo.mismatchFirst = 2

	_, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:    f.tenant,
		MemoryEnabled: boolp(false),
	})
	require.NoError(t, err)
	require.Len(t, f.repo.updates, 1)
	assert.False(t, f.repo.updates[0].MemoryEnabled)
}

func TestUpdateSettings_GivesUpAfterRepeatedConflicts(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	f.repo.mismatchFirst = maxSettingsWriteAttempts

	_, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:    f.tenant,
		MemoryEnabled: boolp(false),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err))
	assert.Empty(t, f.repo.updates)
}

func TestUpdateSettings_RefusesABudgetOutOfRange(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	negative := decimal.RequireFromString("-1")

	_, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:               f.tenant,
		MonthlyIndexingBudgetUSD: &negative,
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Empty(t, f.repo.updates)
}

func TestUpdateSettings_PausingIsManualAndDoesNotWakeTheIndexer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)

	status, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo: f.tenant,
		Paused:     boolp(true),
	})
	require.NoError(t, err)

	require.Len(t, f.repo.updates, 1)
	assert.True(t, f.repo.updates[0].Paused)
	assert.Equal(t, airetrieval.PauseReasonManual, f.repo.updates[0].PausedReason)
	require.NotNil(t, f.repo.updates[0].PausedAt)
	assert.Equal(t, testNow, *f.repo.updates[0].PausedAt)
	require.NotNil(t, status.Settings.PausedReason)
	assert.Equal(t, airetrieval.PauseReasonManual, *status.Settings.PausedReason)
	assert.Zero(t, f.pipeline.woken)
}

func TestUpdateSettings_ResumingLiftsABudgetPauseAndWakesTheIndexer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) {
		s.Paused = true
		s.PausedReason = airetrieval.PauseReasonBudget
		s.PausedAt = int64p(testNow - 60)
	})

	status, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo: f.tenant,
		Paused:     boolp(false),
	})
	require.NoError(t, err)

	require.Len(t, f.repo.updates, 1)
	assert.False(t, f.repo.updates[0].Paused)
	assert.Empty(t, f.repo.updates[0].PausedReason)
	assert.Nil(t, f.repo.updates[0].PausedAt)
	assert.Nil(t, status.Settings.PausedReason)
	assert.Equal(t, 1, f.pipeline.woken)
}

func TestUpdateSettings_RaisingTheBudgetWhileBudgetPausedWakesTheIndexer(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) {
		s.Paused = true
		s.PausedReason = airetrieval.PauseReasonBudget
		s.PausedAt = int64p(testNow - 60)
	})
	raised := decimal.RequireFromString("50")

	_, err := f.svc.UpdateSettings(t.Context(), &serviceports.UpdateAIRetrievalSettingsRequest{
		TenantInfo:               f.tenant,
		MonthlyIndexingBudgetUSD: &raised,
	})
	require.NoError(t, err)

	require.Len(t, f.repo.updates, 1)
	assert.True(t, f.repo.updates[0].PausedByBudget(), "the plan lifts a budget pause, not the patch")
	assert.Equal(t, 1, f.pipeline.woken)
}

func TestApplySettingsPatch_PausingOverABudgetPauseMakesItManual(t *testing.T) {
	t.Parallel()

	settings := airetrieval.DefaultSettings(pulid.MustNew("org_"), pulid.MustNew("bu_"))
	settings.Paused = true
	settings.PausedReason = airetrieval.PauseReasonBudget
	settings.PausedAt = int64p(5)

	changed := ApplySettingsPatch(settings, &serviceports.UpdateAIRetrievalSettingsRequest{
		Paused: boolp(true),
	}, testNow)

	assert.True(t, changed)
	assert.Equal(t, airetrieval.PauseReasonManual, settings.PausedReason)
	assert.Equal(t, testNow, *settings.PausedAt)

	assert.False(t, ApplySettingsPatch(settings, &serviceports.UpdateAIRetrievalSettingsRequest{
		Paused: boolp(true),
	}, testNow+10), "pausing what a person already paused changes nothing")
	assert.Equal(t, testNow, *settings.PausedAt)
}

func TestReindex_StartsTheSourcesReindex(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)

	_, err := f.svc.Reindex(t.Context(), &serviceports.ReindexAIRetrievalSourceRequest{
		TenantInfo: f.tenant,
		SourceType: airetrieval.SourceTypeDocument,
		Actor:      &serviceports.RequestActor{UserID: pulid.MustNew("usr_")},
	})
	require.NoError(t, err)

	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeDocument}, f.indexer.reindexed)
	require.Len(t, f.audit.logged, 1)
	assert.Equal(t, airetrieval.SourceTypeDocument, f.audit.logged[0].CurrentState["sourceType"])
}

func TestReindex_RefusesWhatCannotBeIndexed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		settings     func(*airetrieval.Settings)
		availability airetrieval.Availability
	}{
		{
			name:         "source turned off",
			settings:     func(s *airetrieval.Settings) { s.DocumentsEnabled = false },
			availability: airetrieval.Availability{Available: true},
		},
		{
			name:         "extension missing",
			availability: airetrieval.Availability{Reason: airetrieval.UnavailableReasonExtensionMissing},
		},
		{
			name:         "extension too old",
			availability: airetrieval.Availability{Reason: airetrieval.UnavailableReasonTooOld},
		},
		{
			name:         "schema missing",
			availability: airetrieval.Availability{Reason: airetrieval.UnavailableReasonSchemaMissing},
		},
		{
			name:         "no provider",
			availability: airetrieval.Availability{Reason: airetrieval.UnavailableReasonNoProvider},
		},
		{
			name: "nothing indexed yet",
			settings: func(s *airetrieval.Settings) {
				s.ActiveModelKey = ""
				s.Dimensions = 0
			},
			availability: airetrieval.Availability{Reason: airetrieval.UnavailableReasonNotIndexed},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			f := newFixture(t)
			f.withSettings(tc.settings)
			f.vectorizer.availability = tc.availability

			_, err := f.svc.Reindex(t.Context(), &serviceports.ReindexAIRetrievalSourceRequest{
				TenantInfo: f.tenant,
				SourceType: airetrieval.SourceTypeDocument,
			})

			require.Error(t, err)
			assert.True(t, errortypes.IsBusinessError(err), "got %v", err)
			assert.Empty(t, f.indexer.reindexed)
			assert.Empty(t, f.audit.logged)
		})
	}
}

func TestReindex_WaitsOutAPauseRatherThanRefusing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) {
		s.Paused = true
		s.PausedReason = airetrieval.PauseReasonManual
	})
	f.vectorizer.availability = airetrieval.Availability{Reason: airetrieval.UnavailableReasonDisabled}

	_, err := f.svc.Reindex(t.Context(), &serviceports.ReindexAIRetrievalSourceRequest{
		TenantInfo: f.tenant,
		SourceType: airetrieval.SourceTypeMemory,
	})
	require.NoError(t, err)
	assert.Equal(t, []airetrieval.SourceType{airetrieval.SourceTypeMemory}, f.indexer.reindexed)
}

func TestReindex_RefusesAnUnknownSourceType(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.Reindex(t.Context(), &serviceports.ReindexAIRetrievalSourceRequest{
		TenantInfo: f.tenant,
		SourceType: "Shipments",
	})

	require.Error(t, err)
	assert.Empty(t, f.indexer.reindexed)
}

func TestEstimateReindex(t *testing.T) {
	t.Parallel()

	price := decimal.RequireFromString("0.06")

	t.Run("measured chunks multiply out at the provider's price", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType:     airetrieval.SourceTypeDocument,
			Sources:        110,
			Skipped:        10,
			AverageChars:   4000,
			MeasuredChunks: repositories.IndexChunkAverage{Entries: 90, AverageChunks: 4},
			InputPrice:     &price,
			HasModel:       true,
		})

		assert.Equal(t, 100, got.Sources)
		assert.True(t, got.ChunksMeasured)
		assert.InDelta(t, 4, got.AverageChunks, 0.001)
		assert.InDelta(t, 287.5+ChunkHeaderTokens, got.AverageTokensPerChunk, 0.05)
		assert.Equal(t, 124_600, got.EstimatedTokens)
		require.NotNil(t, got.CostUSD)
		assert.True(t, decimal.RequireFromString("0.007476").Equal(*got.CostUSD), got.CostUSD.String())
	})

	t.Run("unmeasured documents assume the chunker's windows", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType:   airetrieval.SourceTypeDocument,
			Sources:      10,
			AverageChars: 7000,
			InputPrice:   &price,
			HasModel:     true,
		})

		assert.False(t, got.ChunksMeasured)
		assert.InDelta(t, 6, got.AverageChunks, 0.001)
		assert.LessOrEqual(t, got.AverageTokensPerChunk, float64(350+ChunkHeaderTokens))
	})

	t.Run("a memory is one chunk", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType:   airetrieval.SourceTypeMemory,
			Sources:      3,
			AverageChars: 400,
			HasModel:     true,
		})

		assert.InDelta(t, 1, got.AverageChunks, 0.001)
		assert.InDelta(t, 100+MemoryHeaderTokens, got.AverageTokensPerChunk, 0.05)
		assert.Equal(t, 324, got.EstimatedTokens)
		assert.Nil(t, got.CostUSD, "an unpriced provider has an unknown cost, never zero")
	})

	t.Run("measured chunks never exceed the chunker's cap", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType:     airetrieval.SourceTypeInboundMessage,
			Sources:        1,
			AverageChars:   100,
			MeasuredChunks: repositories.IndexChunkAverage{Entries: 1, AverageChunks: 90},
			HasModel:       true,
		})

		assert.InDelta(t, 20, got.AverageChunks, 0.001)
	})

	t.Run("no model means no cost", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType:   airetrieval.SourceTypeMemory,
			Sources:      3,
			AverageChars: 400,
			InputPrice:   &price,
		})

		assert.Nil(t, got.CostUSD)
	})

	t.Run("more skipped than sources is none", func(t *testing.T) {
		t.Parallel()

		got := EstimateReindex(EstimateInputs{
			SourceType: airetrieval.SourceTypeMemory,
			Sources:    2,
			Skipped:    5,
			HasModel:   true,
		})

		assert.Zero(t, got.Sources)
		assert.Zero(t, got.EstimatedTokens)
	})
}

func embeddingProvider(model string, price *decimal.Decimal, enabled bool) *aiprovider.Provider {
	dimensions := 1024

	return &aiprovider.Provider{
		ID:                  pulid.MustNew("aip_"),
		Kind:                aiprovider.KindOpenAIChat,
		BaseURL:             "https://api.voyageai.com/v1",
		Model:               model,
		Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
		Enabled:             enabled,
		EmbeddingDimensions: &dimensions,
		InputCostPerMillion: price,
	}
}

func TestReindexEstimate_PricesWithTheProviderServingTheModel(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	other := decimal.RequireFromString("9")
	price := decimal.RequireFromString("0.06")
	serving := embeddingProvider("voyage-3.5", &price, true)
	f.providers.providers = []*aiprovider.Provider{
		embeddingProvider("voyage-3.5", &other, false),
		embeddingProvider("voyage-3-lite", &other, true),
		serving,
	}
	key := serving.EmbeddingModelKey()
	f.embeddings.key = key
	f.withSettings(func(s *airetrieval.Settings) { s.ActiveModelKey = key })
	f.sources.totals[airetrieval.SourceTypeMemory] = 12
	f.sources.chars = map[airetrieval.SourceType]float64{airetrieval.SourceTypeMemory: 400}
	f.repo.counts[key] = []repositories.IndexEntryCount{
		entryCount(airetrieval.SourceTypeMemory, airetrieval.IndexStatusSkipped, 2),
		entryCount(airetrieval.SourceTypeDocument, airetrieval.IndexStatusSkipped, 7),
	}
	f.repo.averages = map[airetrieval.SourceType]repositories.IndexChunkAverage{
		airetrieval.SourceTypeMemory: {Entries: 8, AverageChunks: 1},
	}
	f.usage.costs[aiusage.SurfaceIndexing] = &repositories.AIUsageCost{
		CostUSD: decimal.RequireFromString("3.25"),
	}

	estimate, err := f.svc.ReindexEstimate(t.Context(), f.tenant, airetrieval.SourceTypeMemory)
	require.NoError(t, err)

	require.NotNil(t, estimate.ModelKey)
	assert.Equal(t, key, *estimate.ModelKey)
	assert.Equal(t, 10, estimate.Sources)
	assert.True(t, estimate.ChunksMeasured)
	assert.Equal(t, 1080, estimate.EstimatedTokens)
	require.NotNil(t, estimate.InputCostPerMillionUSD)
	assert.Equal(t, "0.06", *estimate.InputCostPerMillionUSD)
	require.NotNil(t, estimate.EstimatedCostUSD)
	assert.Equal(t, "0.000065", *estimate.EstimatedCostUSD)
	assert.Equal(t, "6.75", estimate.RemainingBudgetUSD)
}

func TestReindexEstimate_WithoutAProviderHasNoCost(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.embeddings.err = serviceports.ErrNoProviderConfigured
	f.usage.costs[aiusage.SurfaceIndexing] = &repositories.AIUsageCost{
		CostUSD: decimal.RequireFromString("25"),
	}

	estimate, err := f.svc.ReindexEstimate(t.Context(), f.tenant, airetrieval.SourceTypeDocument)
	require.NoError(t, err)

	assert.Nil(t, estimate.ModelKey)
	assert.Nil(t, estimate.EstimatedCostUSD)
	assert.Nil(t, estimate.InputCostPerMillionUSD)
	assert.Equal(
		t,
		"0.00",
		estimate.RemainingBudgetUSD,
		"an overspent budget has nothing left, never less",
	)
}

func erroredEntry(
	sourceType airetrieval.SourceType,
	status airetrieval.IndexStatus,
	attempts int,
	message string,
	lastAttempt int64,
) *airetrieval.IndexEntry {
	return &airetrieval.IndexEntry{
		SourceType:    sourceType,
		SourceID:      pulid.MustNew("src_"),
		ModelKey:      activeKey,
		Status:        status,
		Attempts:      attempts,
		LastError:     message,
		LastAttemptAt: int64p(lastAttempt),
	}
}

func TestListFailedEntries_ReadsTheIndexedModelsAndPages(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(func(s *airetrieval.Settings) {
		s.PendingModelKey = pendingKey
		s.PendingDimensions = 1536
	})
	f.repo.errored = []*airetrieval.IndexEntry{
		erroredEntry(airetrieval.SourceTypeDocument, airetrieval.IndexStatusFailed, 8, someFailure, 300),
		erroredEntry(airetrieval.SourceTypeDocument, airetrieval.IndexStatusPending, 2, "timed out", 200),
		erroredEntry(airetrieval.SourceTypeDocument, airetrieval.IndexStatusFailed, 1, "wrong size", 100),
	}

	page, err := f.svc.ListFailedEntries(
		t.Context(),
		&serviceports.ListAIRetrievalFailedEntriesRequest{
			TenantInfo: f.tenant,
			SourceType: airetrieval.SourceTypeDocument,
			Table:      memtable.Request{First: 2, IncludeTotalCount: true},
		},
	)
	require.NoError(t, err)

	require.NotNil(t, f.repo.erroredReq)
	assert.Equal(t, airetrieval.SourceTypeDocument, f.repo.erroredReq.SourceType)
	assert.Equal(t, []string{activeKey, pendingKey}, f.repo.erroredReq.ModelKeys)
	assert.Equal(t, MaxFailedEntries, f.repo.erroredReq.Limit)

	require.Len(t, page.Edges, 2)
	assert.True(t, page.HasNextPage)
	require.NotNil(t, page.TotalCount)
	assert.Equal(t, 3, *page.TotalCount)
	first := page.Edges[0].Node
	assert.Equal(t, someFailure, first.Error)
	assert.Equal(t, 8, first.Attempts)
	assert.Equal(t, FailedEntryID(f.repo.errored[0]), first.ID)
	assert.NotEqual(t, page.Edges[0].Cursor, page.Edges[1].Cursor)
}

func TestListFailedEntries_CountsOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	f.repo.errored = []*airetrieval.IndexEntry{
		erroredEntry(airetrieval.SourceTypeMemory, airetrieval.IndexStatusFailed, 8, someFailure, 1),
	}

	page, err := f.svc.ListFailedEntries(
		t.Context(),
		&serviceports.ListAIRetrievalFailedEntriesRequest{
			TenantInfo: f.tenant,
		},
	)
	require.NoError(t, err)

	assert.Nil(t, page.TotalCount)
	require.Len(t, page.Edges, 1)
	assert.Empty(t, f.repo.erroredReq.SourceType)
}

func TestListFailedEntries_SearchesTheErrorText(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withSettings(nil)
	f.repo.errored = []*airetrieval.IndexEntry{
		erroredEntry(airetrieval.SourceTypeMemory, airetrieval.IndexStatusFailed, 8, someFailure, 2),
		erroredEntry(
			airetrieval.SourceTypeMemory,
			airetrieval.IndexStatusFailed,
			8,
			"dimension mismatch",
			1,
		),
	}

	page, err := f.svc.ListFailedEntries(
		t.Context(),
		&serviceports.ListAIRetrievalFailedEntriesRequest{
			TenantInfo: f.tenant,
			Table:      memtable.Request{Query: "dimension", IncludeTotalCount: true},
		},
	)
	require.NoError(t, err)

	require.Len(t, page.Edges, 1)
	assert.Equal(t, "dimension mismatch", page.Edges[0].Node.Error)
	assert.Equal(t, 1, *page.TotalCount)
}

func TestListFailedEntries_WithoutAModelListsNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	page, err := f.svc.ListFailedEntries(
		t.Context(),
		&serviceports.ListAIRetrievalFailedEntriesRequest{
			TenantInfo: f.tenant,
			Table:      memtable.Request{IncludeTotalCount: true},
		},
	)
	require.NoError(t, err)

	assert.Nil(t, f.repo.erroredReq, "no model key means no entries to read")
	assert.Empty(t, page.Edges)
	require.NotNil(t, page.TotalCount)
	assert.Zero(t, *page.TotalCount)
}

func TestListFailedEntries_RefusesAnUnknownSourceType(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	_, err := f.svc.ListFailedEntries(t.Context(), &serviceports.ListAIRetrievalFailedEntriesRequest{
		TenantInfo: f.tenant,
		SourceType: "Shipments",
	})

	require.Error(t, err)
	assert.Nil(t, f.repo.erroredReq)
}
