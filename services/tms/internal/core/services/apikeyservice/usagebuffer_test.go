package apikeyservice

import (
	"context"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestUsageBuffer(t *testing.T, repo *mocks.MockAPIKeyRepository) *UsageBuffer {
	t.Helper()

	return NewUsageBuffer(repo, zap.NewNop(), &config.Config{
		Security: config.SecurityConfig{
			APIToken: config.APITokenConfig{
				UsageFlushInterval: 25 * time.Millisecond,
				UsageUpdateTimeout: time.Second,
				UsageMaxPending:    32,
			},
		},
	})
}

func TestUsageBufferAggregatesSameKeyAndFlushesMetadata(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	occurredAt := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)

	for range 5 {
		buf.RecordUsage(services.APIKeyUsageEvent{
			APIKeyID:       keyID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
			OccurredAt:     occurredAt,
			IPAddress:      "192.0.2.10",
			UserAgent:      "integration-test",
		})
	}

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, bucketUsageDate(occurredAt), int64(5),
	).Return(nil).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, repositories.APIKeyUsageMetadata{
		LastUsedAt:        occurredAt.Unix(),
		LastUsedIP:        "192.0.2.10",
		LastUsedUserAgent: "integration-test",
	}).Return(nil).Once()

	require.NoError(t, buf.flush(t.Context()))
	assert.Empty(t, buf.counts)
	assert.Empty(t, buf.lastUsed)
}

func TestUsageBufferUsesLatestMetadataPerAPIKey(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	firstSeen := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)
	laterSeen := firstSeen.Add(2 * time.Minute)
	longAgent := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0123456789"
	expectedAgent := clampString(longAgent, maxUserAgentLength)

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     firstSeen,
		IPAddress:      "192.0.2.10",
		UserAgent:      "older-agent",
	})
	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     laterSeen,
		IPAddress:      "198.51.100.7",
		UserAgent:      longAgent,
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, bucketUsageDate(firstSeen), int64(2),
	).Return(nil).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, mock.MatchedBy(func(
		metadata repositories.APIKeyUsageMetadata,
	) bool {
		return metadata.LastUsedAt == laterSeen.Unix() &&
			metadata.LastUsedIP == "198.51.100.7" &&
			metadata.LastUsedUserAgent == expectedAgent
	})).Return(nil).Once()

	require.NoError(t, buf.flush(t.Context()))
}

func TestUsageBufferFlushRequeuesFailures(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	occurredAt := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)
	expectedDate := bucketUsageDate(occurredAt)
	expectedMetadata := repositories.APIKeyUsageMetadata{
		LastUsedAt:        occurredAt.Unix(),
		LastUsedIP:        "192.0.2.10",
		LastUsedUserAgent: "integration-test",
	}

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "192.0.2.10",
		UserAgent:      "integration-test",
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, expectedDate, int64(1),
	).Return(assert.AnError).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, expectedMetadata).Return(assert.AnError).Once()

	require.Error(t, buf.flush(t.Context()))
	assert.Len(t, buf.counts, 1)
	assert.Len(t, buf.lastUsed, 1)

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, expectedDate, int64(1),
	).Return(nil).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, expectedMetadata).Return(nil).Once()

	require.NoError(t, buf.flush(t.Context()))
	assert.Empty(t, buf.counts)
	assert.Empty(t, buf.lastUsed)
}

func TestUsageBufferStopFlushesRemaining(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	occurredAt := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "192.0.2.10",
		UserAgent:      "integration-test",
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, bucketUsageDate(occurredAt), int64(1),
	).Return(nil).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, repositories.APIKeyUsageMetadata{
		LastUsedAt:        occurredAt.Unix(),
		LastUsedIP:        "192.0.2.10",
		LastUsedUserAgent: "integration-test",
	}).Return(nil).Once()

	buf.Start()
	require.NoError(t, buf.Stop(t.Context()))
	require.NoError(t, buf.Stop(t.Context()))
	assert.Empty(t, buf.counts)
	assert.Empty(t, buf.lastUsed)
}

func TestUsageBufferBucketsDatesInUTC(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	loc := time.FixedZone("UTC-5", -5*60*60)
	occurredAt := time.Date(2026, 3, 6, 23, 30, 0, 0, loc)
	expectedDate := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "192.0.2.10",
		UserAgent:      "integration-test",
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything, keyID, orgID, buID, expectedDate, int64(1),
	).Return(nil).Once()
	repo.EXPECT().UpdateUsage(mock.Anything, keyID, mock.Anything).Return(nil).Once()

	require.NoError(t, buf.flush(t.Context()))
}

func TestUsageBufferDropsNewKeysWhenPendingBufferIsFull(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := NewUsageBuffer(repo, zap.NewNop(), &config.Config{
		Security: config.SecurityConfig{
			APIToken: config.APITokenConfig{
				UsageFlushInterval: time.Second,
				UsageUpdateTimeout: time.Second,
				UsageMaxPending:    2,
			},
		},
	})

	firstEvent := services.APIKeyUsageEvent{
		APIKeyID:       pulid.MustNew("ak_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OccurredAt:     time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC),
	}
	secondEvent := services.APIKeyUsageEvent{
		APIKeyID:       pulid.MustNew("ak_"),
		OrganizationID: firstEvent.OrganizationID,
		BusinessUnitID: firstEvent.BusinessUnitID,
		OccurredAt:     firstEvent.OccurredAt,
	}

	buf.RecordUsage(firstEvent)
	buf.RecordUsage(secondEvent)

	assert.Len(t, buf.counts, 1)
	assert.Len(t, buf.lastUsed, 1)
	_, secondCount := buf.counts[usageKey{
		apiKeyID: secondEvent.APIKeyID,
		date:     bucketUsageDate(secondEvent.OccurredAt),
	}]
	_, secondMetadata := buf.lastUsed[secondEvent.APIKeyID]
	assert.False(t, secondCount)
	assert.False(t, secondMetadata)
}

func TestUsageBufferFlushCancellationRequeuesRemainingItems(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	firstKeyID := pulid.MustNew("ak_")
	secondKeyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	occurredAt := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)
	blocked := make(chan struct{}, 1)

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       firstKeyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "192.0.2.10",
		UserAgent:      "integration-test",
	})
	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       secondKeyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "198.51.100.7",
		UserAgent:      "second-agent",
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).RunAndReturn(func(
		ctx context.Context,
		_ pulid.ID,
		_ pulid.ID,
		_ pulid.ID,
		_ time.Time,
		_ int64,
	) error {
		select {
		case blocked <- struct{}{}:
		default:
		}
		<-ctx.Done()
		return ctx.Err()
	}).Once()

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()

	err := buf.flush(ctx)
	require.Error(t, err)
	<-blocked

	assert.Len(t, buf.counts, 2)
	assert.Len(t, buf.lastUsed, 2)
}

func TestUsageBufferStopCancelsInFlightFlush(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := NewUsageBuffer(repo, zap.NewNop(), &config.Config{
		Security: config.SecurityConfig{
			APIToken: config.APITokenConfig{
				UsageFlushInterval: 10 * time.Millisecond,
				UsageUpdateTimeout: time.Second,
				UsageMaxPending:    8,
			},
		},
	})

	keyID := pulid.MustNew("ak_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	occurredAt := time.Date(2026, 3, 6, 10, 15, 0, 0, time.UTC)
	started := make(chan struct{}, 2)
	finished := make(chan struct{}, 2)

	buf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       keyID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		OccurredAt:     occurredAt,
		IPAddress:      "192.0.2.10",
		UserAgent:      "integration-test",
	})

	repo.EXPECT().IncrementDailyUsage(
		mock.Anything,
		keyID,
		orgID,
		buID,
		bucketUsageDate(occurredAt),
		int64(1),
	).RunAndReturn(func(
		ctx context.Context,
		_ pulid.ID,
		_ pulid.ID,
		_ pulid.ID,
		_ time.Time,
		_ int64,
	) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-ctx.Done()
		select {
		case finished <- struct{}{}:
		default:
		}
		return ctx.Err()
	}).Twice()

	buf.Start()
	<-started

	stopCtx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := buf.Stop(stopCtx)
	elapsed := time.Since(start)
	<-finished
	<-finished

	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, elapsed, 250*time.Millisecond)
}

func TestUsageBufferRetryRequeueHonorsCapacity(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := NewUsageBuffer(repo, zap.NewNop(), &config.Config{
		Security: config.SecurityConfig{
			APIToken: config.APITokenConfig{
				UsageFlushInterval: time.Second,
				UsageUpdateTimeout: time.Second,
				UsageMaxPending:    2,
			},
		},
	})

	existingKeyID := pulid.MustNew("ak_")
	newKeyID := pulid.MustNew("ak_")
	usageDate := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)

	buf.counts[usageKey{apiKeyID: existingKeyID, date: usageDate}] = &usageEntry{
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
		count: 1,
	}
	buf.lastUsed[existingKeyID] = &usageMetadata{
		lastUsedAt:        usageDate.Unix(),
		lastUsedIP:        "192.0.2.1",
		lastUsedUserAgent: "agent-1",
		occurredAt:        usageDate,
	}

	buf.requeueCount(usageKey{apiKeyID: newKeyID, date: usageDate}, &usageEntry{
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
		count: 2,
	}, "repository_error")
	buf.requeueMetadata(newKeyID, &usageMetadata{
		lastUsedAt:        usageDate.Unix(),
		lastUsedIP:        "198.51.100.7",
		lastUsedUserAgent: "agent-2",
		occurredAt:        usageDate,
	}, "repository_error")

	assert.Len(t, buf.counts, 1)
	assert.Len(t, buf.lastUsed, 1)
	_, newCount := buf.counts[usageKey{apiKeyID: newKeyID, date: usageDate}]
	_, newMetadata := buf.lastUsed[newKeyID]
	assert.False(t, newCount)
	assert.False(t, newMetadata)

	buf.requeueCount(usageKey{apiKeyID: existingKeyID, date: usageDate}, &usageEntry{
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
		count: 3,
	}, "repository_error")
	buf.requeueMetadata(existingKeyID, &usageMetadata{
		lastUsedAt:        usageDate.Add(time.Minute).Unix(),
		lastUsedIP:        "203.0.113.8",
		lastUsedUserAgent: "agent-3",
		occurredAt:        usageDate.Add(time.Minute),
	}, "repository_error")

	assert.Equal(t, int64(4), buf.counts[usageKey{apiKeyID: existingKeyID, date: usageDate}].count)
	assert.Equal(t, "203.0.113.8", buf.lastUsed[existingKeyID].lastUsedIP)
}

func TestClampStringPreservesValidUTF8(t *testing.T) {
	t.Parallel()

	value := "truck-" + "配送" + "-alpha"
	truncated := clampString(value, 7)

	assert.True(t, utf8.ValidString(truncated))
	assert.Equal(t, 7, utf8.RuneCountInString(truncated))
	assert.Equal(t, "truck-配", truncated)
}

func TestUsageBufferStopWithoutStartIsNoop(t *testing.T) {
	t.Parallel()

	repo := mocks.NewMockAPIKeyRepository(t)
	buf := newTestUsageBuffer(t, repo)

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	require.NoError(t, buf.Stop(ctx))
}
