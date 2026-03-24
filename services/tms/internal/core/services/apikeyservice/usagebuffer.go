package apikeyservice

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxIPAddressLength = 45
	maxUserAgentLength = 255
)

type usageKey struct {
	apiKeyID pulid.ID
	date     time.Time
}

type usageEntry struct {
	orgID pulid.ID
	buID  pulid.ID
	count int64
}

type usageMetadata struct {
	lastUsedAt        int64
	lastUsedIP        string
	lastUsedUserAgent string
	occurredAt        time.Time
}

type countFlushItem struct {
	key   usageKey
	entry *usageEntry
}

type metadataFlushItem struct {
	apiKeyID pulid.ID
	metadata *usageMetadata
}

type stopRequest struct {
	ctx  context.Context
	done chan error
}

type UsageBuffer struct {
	mu           sync.Mutex
	counts       map[usageKey]*usageEntry
	lastUsed     map[pulid.ID]*usageMetadata
	repo         repositories.APIKeyRepository
	l            *zap.Logger
	flushEvery   time.Duration
	writeTimeout time.Duration
	maxPending   int
	stopCh       chan stopRequest
	startOnce    sync.Once
	stopOnce     sync.Once
	started      atomic.Bool
	workerCtx    context.Context
	cancelFlush  context.CancelFunc
}

type UsageBufferParams struct {
	fx.In

	LC     fx.Lifecycle
	Repo   repositories.APIKeyRepository
	Logger *zap.Logger
	Config *config.Config
}

func NewUsageBuffer(
	repo repositories.APIKeyRepository,
	l *zap.Logger,
	cfg *config.Config,
) *UsageBuffer {
	apiTokenCfg := config.APITokenConfig{}
	if cfg != nil {
		apiTokenCfg = cfg.Security.APIToken
	}

	return &UsageBuffer{
		counts:       make(map[usageKey]*usageEntry),
		lastUsed:     make(map[pulid.ID]*usageMetadata),
		repo:         repo,
		l:            l.Named("usage-buffer"),
		flushEvery:   apiTokenCfg.GetUsageFlushInterval(),
		writeTimeout: apiTokenCfg.GetUsageUpdateTimeout(),
		maxPending:   apiTokenCfg.GetUsageMaxPending(),
		stopCh:       make(chan stopRequest, 1),
	}
}

func NewUsageBufferWithLifecycle(p UsageBufferParams) *UsageBuffer {
	buf := NewUsageBuffer(p.Repo, p.Logger, p.Config)
	p.LC.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			buf.Start()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return buf.Stop(ctx)
		},
	})
	return buf
}

func (b *UsageBuffer) RecordUsage(event services.APIKeyUsageEvent) {
	if event.APIKeyID.IsNil() {
		return
	}

	occurredAt := event.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	key := usageKey{
		apiKeyID: event.APIKeyID,
		date:     bucketUsageDate(occurredAt),
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.canAcceptEventLocked(key, event.APIKeyID) {
		b.l.Warn(
			"dropping api key usage event because pending buffer is full",
			zap.String("apiKeyID", event.APIKeyID.String()),
		)
		return
	}

	entry, ok := b.counts[key]
	if !ok {
		entry = &usageEntry{
			orgID: event.OrganizationID,
			buID:  event.BusinessUnitID,
		}
		b.counts[key] = entry
	}
	entry.count++

	metadata, ok := b.lastUsed[event.APIKeyID]
	if !ok || occurredAt.After(metadata.occurredAt) {
		b.lastUsed[event.APIKeyID] = &usageMetadata{
			lastUsedAt:        occurredAt.Unix(),
			lastUsedIP:        clampString(event.IPAddress, maxIPAddressLength),
			lastUsedUserAgent: clampString(event.UserAgent, maxUserAgentLength),
			occurredAt:        occurredAt,
		}
	}
}

func (b *UsageBuffer) Start() {
	b.startOnce.Do(func() {
		b.started.Store(true)
		b.workerCtx, b.cancelFlush = context.WithCancel(context.Background())

		go func() {
			ticker := time.NewTicker(b.flushEvery)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					_ = b.flush(b.workerCtx)
				case req := <-b.stopCh:
					err := b.flush(req.ctx)
					req.done <- err
					close(req.done)
					return
				}
			}
		}()
	})
}

func (b *UsageBuffer) Stop(ctx context.Context) error {
	if !b.started.Load() {
		return nil
	}

	var stopErr error
	b.stopOnce.Do(func() {
		if b.cancelFlush != nil {
			b.cancelFlush()
		}

		done := make(chan error, 1)
		b.stopCh <- stopRequest{ctx: ctx, done: done}

		select {
		case err := <-done:
			stopErr = err
		case <-ctx.Done():
			stopErr = ctx.Err()
		}
	})

	return stopErr
}

func (b *UsageBuffer) flush(parent context.Context) error {
	countsSnapshot, metadataSnapshot := b.snapshot()
	if len(countsSnapshot) == 0 && len(metadataSnapshot) == 0 {
		return nil
	}

	var firstErr error

	for i, item := range countsSnapshot {
		if err := parent.Err(); err != nil {
			b.requeueRemainingCounts(countsSnapshot[i:])
			b.requeueRemainingMetadata(metadataSnapshot)
			return firstNonNil(firstErr, err)
		}

		ctx, cancel := context.WithTimeout(parent, b.writeTimeout)
		err := b.repo.IncrementDailyUsage(
			ctx,
			item.key.apiKeyID,
			item.entry.orgID,
			item.entry.buID,
			item.key.date,
			item.entry.count,
		)
		cancel()
		if err != nil {
			if parent.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				b.requeueCount(item.key, item.entry, "canceled")
				b.requeueRemainingCounts(countsSnapshot[i+1:])
				b.requeueRemainingMetadata(metadataSnapshot)
				return firstNonNil(firstErr, err)
			}
			if firstErr == nil {
				firstErr = err
			}
			b.requeueCount(item.key, item.entry, "repository_error", zap.Error(err))
		}
	}

	for i, item := range metadataSnapshot {
		if err := parent.Err(); err != nil {
			b.requeueRemainingMetadata(metadataSnapshot[i:])
			return firstNonNil(firstErr, err)
		}

		ctx, cancel := context.WithTimeout(parent, b.writeTimeout)
		err := b.repo.UpdateUsage(ctx, item.apiKeyID, repositories.APIKeyUsageMetadata{
			LastUsedAt:        item.metadata.lastUsedAt,
			LastUsedIP:        item.metadata.lastUsedIP,
			LastUsedUserAgent: item.metadata.lastUsedUserAgent,
		})
		cancel()
		if err != nil {
			if parent.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				b.requeueMetadata(item.apiKeyID, item.metadata, "canceled")
				b.requeueRemainingMetadata(metadataSnapshot[i+1:])
				return firstNonNil(firstErr, err)
			}
			if firstErr == nil {
				firstErr = err
			}
			b.requeueMetadata(item.apiKeyID, item.metadata, "repository_error", zap.Error(err))
		}
	}

	return firstErr
}

func (b *UsageBuffer) snapshot() ([]countFlushItem, []metadataFlushItem) {
	b.mu.Lock()
	defer b.mu.Unlock()

	countsSnapshot := b.counts
	metadataSnapshot := b.lastUsed
	b.counts = make(map[usageKey]*usageEntry, len(countsSnapshot))
	b.lastUsed = make(map[pulid.ID]*usageMetadata, len(metadataSnapshot))

	countItems := make([]countFlushItem, 0, len(countsSnapshot))
	for key, entry := range countsSnapshot {
		countItems = append(countItems, countFlushItem{key: key, entry: entry})
	}

	metadataItems := make([]metadataFlushItem, 0, len(metadataSnapshot))
	for apiKeyID, metadata := range metadataSnapshot {
		metadataItems = append(metadataItems, metadataFlushItem{
			apiKeyID: apiKeyID,
			metadata: metadata,
		})
	}

	return countItems, metadataItems
}

func (b *UsageBuffer) requeueCount(key usageKey, entry *usageEntry, reason string, fields ...zap.Field) {
	b.mu.Lock()
	defer b.mu.Unlock()

	existing, ok := b.counts[key]
	if !ok {
		if !b.canAcceptCountLocked(key) {
			b.logDrop("api key usage count", key.apiKeyID, reason, fields...)
			return
		}
		b.counts[key] = entry
		b.logRequeue("api key usage count", key.apiKeyID, reason, entry.count, fields...)
		return
	}

	existing.count += entry.count
	b.logRequeue("api key usage count", key.apiKeyID, reason, entry.count, fields...)
}

func (b *UsageBuffer) requeueMetadata(
	apiKeyID pulid.ID,
	metadata *usageMetadata,
	reason string,
	fields ...zap.Field,
) {
	b.mu.Lock()
	defer b.mu.Unlock()

	existing, ok := b.lastUsed[apiKeyID]
	if !ok {
		if !b.canAcceptMetadataLocked(apiKeyID) {
			b.logDrop("api key last-used metadata", apiKeyID, reason, fields...)
			return
		}
		b.lastUsed[apiKeyID] = metadata
		b.logRequeue("api key last-used metadata", apiKeyID, reason, 0, fields...)
		return
	}

	if metadata.occurredAt.After(existing.occurredAt) {
		b.lastUsed[apiKeyID] = metadata
	}
	b.logRequeue("api key last-used metadata", apiKeyID, reason, 0, fields...)
}

func (b *UsageBuffer) requeueRemainingCounts(items []countFlushItem) {
	for _, item := range items {
		b.requeueCount(item.key, item.entry, "canceled")
	}
}

func (b *UsageBuffer) requeueRemainingMetadata(items []metadataFlushItem) {
	for _, item := range items {
		b.requeueMetadata(item.apiKeyID, item.metadata, "canceled")
	}
}

func (b *UsageBuffer) canAcceptEventLocked(key usageKey, apiKeyID pulid.ID) bool {
	if b.maxPending <= 0 {
		return true
	}

	requiredSlots := 0
	if _, ok := b.counts[key]; !ok {
		requiredSlots++
	}
	if _, ok := b.lastUsed[apiKeyID]; !ok {
		requiredSlots++
	}

	return len(b.counts)+len(b.lastUsed)+requiredSlots <= b.maxPending
}

func (b *UsageBuffer) canAcceptCountLocked(key usageKey) bool {
	if b.maxPending <= 0 {
		return true
	}

	if _, ok := b.counts[key]; ok {
		return true
	}

	return len(b.counts)+len(b.lastUsed) < b.maxPending
}

func (b *UsageBuffer) canAcceptMetadataLocked(apiKeyID pulid.ID) bool {
	if b.maxPending <= 0 {
		return true
	}

	if _, ok := b.lastUsed[apiKeyID]; ok {
		return true
	}

	return len(b.counts)+len(b.lastUsed) < b.maxPending
}

func bucketUsageDate(t time.Time) time.Time {
	return t.UTC().Truncate(24 * time.Hour)
}

func clampString(value string, maxLen int) string {
	if maxLen <= 0 || value == "" {
		return ""
	}

	if utf8.RuneCountInString(value) <= maxLen {
		return value
	}

	runes := []rune(value)
	return string(runes[:maxLen])
}

func firstNonNil(primary, fallback error) error {
	if primary != nil {
		return primary
	}
	return fallback
}

func (b *UsageBuffer) logRequeue(
	message string,
	apiKeyID pulid.ID,
	reason string,
	count int64,
	fields ...zap.Field,
) {
	logFields := []zap.Field{
		zap.String("apiKeyID", apiKeyID.String()),
		zap.String("reason", reason),
	}
	if count > 0 {
		logFields = append(logFields, zap.Int64("count", count))
	}
	logFields = append(logFields, fields...)
	b.l.Warn("re-enqueueing "+message, logFields...)
}

func (b *UsageBuffer) logDrop(
	message string,
	apiKeyID pulid.ID,
	reason string,
	fields ...zap.Field,
) {
	logFields := []zap.Field{
		zap.String("apiKeyID", apiKeyID.String()),
		zap.String("reason", reason),
	}
	logFields = append(logFields, fields...)
	b.l.Warn("dropping "+message+" because pending buffer is full", logFields...)
}
