package services

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
)

type accountingSyncCollectorKey struct{}

type AccountingSyncCollector struct {
	mu      sync.Mutex
	records []*accountingsync.AccountingSyncRecord
}

func CollectAccountingSync(ctx context.Context) (context.Context, *AccountingSyncCollector) {
	collector := new(AccountingSyncCollector)
	return context.WithValue(ctx, accountingSyncCollectorKey{}, collector), collector
}

func NoteAccountingSyncEnqueued(
	ctx context.Context,
	records []*accountingsync.AccountingSyncRecord,
) {
	collector, ok := ctx.Value(accountingSyncCollectorKey{}).(*AccountingSyncCollector)
	if !ok || collector == nil || len(records) == 0 {
		return
	}
	collector.mu.Lock()
	defer collector.mu.Unlock()
	collector.records = append(collector.records, records...)
}

func (c *AccountingSyncCollector) Records() []*accountingsync.AccountingSyncRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]*accountingsync.AccountingSyncRecord, len(c.records))
	copy(out, c.records)
	return out
}
