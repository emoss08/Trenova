package retrievalquery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newVectorizer(
	repo *fakeRepository,
	embeddings serviceports.EmbeddingService,
) *Service {
	return New(Params{Logger: zap.NewNop(), Repository: repo, Embeddings: embeddings})
}

func queryRequest(tenant pagination.TenantInfo, text string) *serviceports.QueryVectorRequest {
	return &serviceports.QueryVectorRequest{
		TenantInfo: tenant,
		Text:       text,
		Attribution: serviceports.AIUsageAttribution{
			UserID:   pulid.MustNew("usr_"),
			ThreadID: pulid.MustNew("ath_"),
		},
	}
}

func TestVectorize_EmbedsTheQueryUnderTheActiveModel(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(3) }}
	service := newVectorizer(newFakeRepository(activeSettings(tenant)), embeddings)

	req := queryRequest(tenant, "  who is off this week  ")
	vector, err := service.Vectorize(t.Context(), req)
	require.NoError(t, err)

	require.True(t, vector.Usable())
	assert.Equal(t, testModelKey, vector.ModelKey)
	assert.Equal(t, airetrieval.Dimensions768, vector.Dimensions)
	assert.Empty(t, vector.Reason)

	calls := embeddings.calls()
	require.Len(t, calls, 1)
	assert.Equal(t, serviceports.EmbeddingPurposeQuery, calls[0].Purpose)
	assert.Equal(t, testModelKey, calls[0].ModelKey, "the query is pinned to the index's model")
	assert.Equal(t, []string{"who is off this week"}, calls[0].Inputs)
	assert.Zero(t, calls[0].QueryTimeout, "the router's 1.5 s default applies")
	assert.Equal(t, serviceports.DefaultQueryEmbeddingTimeout, calls[0].ResolvedQueryTimeout())
	assert.Equal(t, req.Attribution, calls[0].Attribution)
	assert.Equal(t, tenant, calls[0].TenantInfo)
}

func TestVectorize_EmbedsATextOncePerTenantAndModel(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(1) }}
	repo := newFakeRepository(activeSettings(tenant))
	service := newVectorizer(repo, embeddings)

	for range 3 {
		_, err := service.Vectorize(t.Context(), queryRequest(tenant, "chase the shipper"))
		require.NoError(t, err)
	}
	require.Len(t, embeddings.calls(), 1, "a turn's callers share one embedding")

	other := testTenant()
	repo.settings = activeSettings(other)
	_, err := service.Vectorize(t.Context(), queryRequest(other, "chase the shipper"))
	require.NoError(t, err)
	assert.Len(t, embeddings.calls(), 2, "another organization's vector is its own")
}

func TestVectorize_ReportsWhyRetrievalIsUnavailable(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	paused := activeSettings(tenant)
	paused.Paused = true
	paused.PausedReason = airetrieval.PauseReasonManual
	budget := activeSettings(tenant)
	budget.Paused = true
	budget.PausedReason = airetrieval.PauseReasonBudget
	unindexed := airetrieval.DefaultSettings(tenant.OrgID, tenant.BuID)

	cases := []struct {
		name       string
		repo       func() *fakeRepository
		embeddings func() *fakeEmbeddings
		want       airetrieval.UnavailableReason
	}{
		{
			name: "the extension is missing",
			repo: func() *fakeRepository {
				repo := newFakeRepository(activeSettings(tenant))
				repo.availability = airetrieval.Availability{
					Reason: airetrieval.UnavailableReasonExtensionMissing,
				}
				return repo
			},
			want: airetrieval.UnavailableReasonExtensionMissing,
		},
		{
			name: "the schema is missing",
			repo: func() *fakeRepository {
				repo := newFakeRepository(activeSettings(tenant))
				repo.availability = airetrieval.Availability{
					Reason:             airetrieval.UnavailableReasonSchemaMissing,
					ExtensionInstalled: true,
				}
				return repo
			},
			want: airetrieval.UnavailableReasonSchemaMissing,
		},
		{
			name: "retrieval is paused by hand",
			repo: func() *fakeRepository { return newFakeRepository(paused) },
			want: airetrieval.UnavailableReasonDisabled,
		},
		{
			name: "the indexing budget is spent",
			repo: func() *fakeRepository { return newFakeRepository(budget) },
			want: airetrieval.UnavailableReasonBudgetPaused,
		},
		{
			name: "nothing is indexed yet",
			repo: func() *fakeRepository { return newFakeRepository(unindexed) },
			want: airetrieval.UnavailableReasonNotIndexed,
		},
		{
			name: "no provider embeds and nothing is indexed",
			repo: func() *fakeRepository { return newFakeRepository(unindexed) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{configured: serviceports.ErrNoProviderConfigured}
			},
			want: airetrieval.UnavailableReasonNoProvider,
		},
		{
			name: "no provider serves the indexed model",
			repo: func() *fakeRepository { return newFakeRepository(activeSettings(tenant)) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{
					err: fmt.Errorf("route: %w", serviceports.ErrNoProviderConfigured),
				}
			},
			want: airetrieval.UnavailableReasonNoProvider,
		},
		{
			name: "the provider is too slow",
			repo: func() *fakeRepository { return newFakeRepository(activeSettings(tenant)) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{err: fmt.Errorf("embed: %w", context.DeadlineExceeded)}
			},
			want: airetrieval.UnavailableReasonQueryTimeout,
		},
		{
			name: "the provider fails",
			repo: func() *fakeRepository { return newFakeRepository(activeSettings(tenant)) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{err: errors.New("502 bad gateway")}
			},
			want: airetrieval.UnavailableReasonProviderFailed,
		},
		{
			name: "the provider answers with the wrong size",
			repo: func() *fakeRepository { return newFakeRepository(activeSettings(tenant)) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{vectorFor: func(string) []float32 { return []float32{1, 2} }}
			},
			want: airetrieval.UnavailableReasonProviderFailed,
		},
		{
			name: "the provider answers under another model",
			repo: func() *fakeRepository { return newFakeRepository(activeSettings(tenant)) },
			embeddings: func() *fakeEmbeddings {
				return &fakeEmbeddings{
					vectorFor: func(string) []float32 { return axis(0) },
					modelKey:  "api.openai.com/text-embedding-3-small@768",
				}
			},
			want: airetrieval.UnavailableReasonProviderFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(0) }}
			if tc.embeddings != nil {
				embeddings = tc.embeddings()
			}
			service := newVectorizer(tc.repo(), embeddings)

			vector, err := service.Vectorize(t.Context(), queryRequest(tenant, "who is off"))
			require.NoError(t, err, "an unavailable vector is a fallback, not a failure")
			assert.False(t, vector.Available)
			assert.False(t, vector.Usable())
			assert.Equal(t, tc.want, vector.Reason)
		})
	}
}

func TestVectorize_WithoutAnEmbeddingServiceHasNoProvider(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	service := newVectorizer(newFakeRepository(activeSettings(tenant)), nil)

	vector, err := service.Vectorize(t.Context(), queryRequest(tenant, "who is off"))
	require.NoError(t, err)
	assert.Equal(t, airetrieval.UnavailableReasonNoProvider, vector.Reason)
}

func TestVectorize_DoesNotCacheAFallback(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	embeddings := &fakeEmbeddings{err: context.DeadlineExceeded}
	service := newVectorizer(newFakeRepository(activeSettings(tenant)), embeddings)

	first, err := service.Vectorize(t.Context(), queryRequest(tenant, "who is off"))
	require.NoError(t, err)
	require.Equal(t, airetrieval.UnavailableReasonQueryTimeout, first.Reason)

	embeddings.err = nil
	embeddings.vectorFor = func(string) []float32 { return axis(2) }
	second, err := service.Vectorize(t.Context(), queryRequest(tenant, "who is off"))
	require.NoError(t, err)
	assert.True(t, second.Usable())
}

func TestVectorize_ReturnsWhatACallerMustSee(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(0) }}

	service := newVectorizer(newFakeRepository(activeSettings(tenant)), embeddings)
	_, err := service.Vectorize(t.Context(), queryRequest(tenant, "   "))
	require.ErrorIs(t, err, serviceports.ErrQueryTextRequired)

	_, err = service.Vectorize(t.Context(), queryRequest(pagination.TenantInfo{}, "who"))
	require.ErrorIs(t, err, serviceports.ErrQueryTenantRequired)

	broken := newFakeRepository(nil)
	broken.settingsErr = errors.New("connection refused")
	_, err = newVectorizer(broken, embeddings).Vectorize(t.Context(), queryRequest(tenant, "who"))
	require.Error(t, err)

	probe := newFakeRepository(activeSettings(tenant))
	probe.availErr = errors.New("probe failed")
	_, err = newVectorizer(probe, embeddings).Vectorize(t.Context(), queryRequest(tenant, "who"))
	require.Error(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	failing := &fakeEmbeddings{err: context.Canceled}
	_, err = newVectorizer(newFakeRepository(activeSettings(tenant)), failing).
		Vectorize(ctx, queryRequest(tenant, "who"))
	require.ErrorIs(t, err, context.Canceled)

	assert.Empty(t, embeddings.calls())
}

func TestVectorize_BoundsTheTextItSends(t *testing.T) {
	t.Parallel()

	tenant := testTenant()
	embeddings := &fakeEmbeddings{vectorFor: func(string) []float32 { return axis(0) }}
	service := newVectorizer(newFakeRepository(activeSettings(tenant)), embeddings)

	_, err := service.Vectorize(
		t.Context(),
		queryRequest(tenant, strings.Repeat("é", serviceports.MaxQueryTextRunes+50)),
	)
	require.NoError(t, err)

	calls := embeddings.calls()
	require.Len(t, calls, 1)
	assert.Len(t, []rune(calls[0].Inputs[0]), serviceports.MaxQueryTextRunes)
}

func TestAvailability(t *testing.T) {
	t.Parallel()

	tenant := testTenant()

	ready := newVectorizer(
		newFakeRepository(activeSettings(tenant)),
		&fakeEmbeddings{},
	)
	availability, err := ready.Availability(t.Context(), tenant)
	require.NoError(t, err)
	assert.True(t, availability.Available)

	unindexed := newVectorizer(
		newFakeRepository(airetrieval.DefaultSettings(tenant.OrgID, tenant.BuID)),
		&fakeEmbeddings{},
	)
	availability, err = unindexed.Availability(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, airetrieval.UnavailableReasonNotIndexed, availability.Reason)
	assert.True(t, availability.ExtensionInstalled)

	unprovided := newVectorizer(
		newFakeRepository(activeSettings(tenant)),
		&fakeEmbeddings{configured: serviceports.ErrNoProviderConfigured},
	)
	availability, err = unprovided.Availability(t.Context(), tenant)
	require.NoError(t, err)
	assert.Equal(t, airetrieval.UnavailableReasonNoProvider, availability.Reason)

	broken := newVectorizer(
		newFakeRepository(activeSettings(tenant)),
		&fakeEmbeddings{configured: errors.New("database is down")},
	)
	_, err = broken.Availability(t.Context(), tenant)
	require.Error(t, err)

	_, err = ready.Availability(t.Context(), pagination.TenantInfo{})
	require.ErrorIs(t, err, serviceports.ErrQueryTenantRequired)
}
