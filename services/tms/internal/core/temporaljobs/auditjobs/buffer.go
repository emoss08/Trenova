package auditjobs

import (
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"go.uber.org/atomic"
)

type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type Buffer struct {
	entries      []*audit.Entry
	mu           sync.RWMutex
	limit        int
	failureCount *atomic.Int64
	circuitState atomic.Int32
	lastFailure  atomic.Int64
	cooldownTime time.Duration
}

func NewBuffer(limit int) *Buffer {
	return &Buffer{
		entries:      make([]*audit.Entry, 0, limit),
		limit:        limit,
		failureCount: atomic.NewInt64(0),
		circuitState: *atomic.NewInt32(int32(CircuitClosed)),
		lastFailure:  *atomic.NewInt64(0),
		cooldownTime: 30 * time.Second,
	}
}

func (b *Buffer) Add(entry *audit.Entry) bool {
	currentState := CircuitState(b.circuitState.Load())
	if currentState == CircuitOpen {
		lastFailureTime := time.Unix(b.lastFailure.Load(), 0)
		if time.Since(lastFailureTime) > b.cooldownTime {
			b.circuitState.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen))
		} else {
			return false
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) >= b.limit {
		return false
	}

	b.entries = append(b.entries, entry)

	if CircuitState(b.circuitState.Load()) == CircuitHalfOpen {
		b.circuitState.Store(int32(CircuitClosed))
		b.failureCount.Store(0)
	}

	return true
}

func (b *Buffer) FlushAndReset() []*audit.Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		return nil
	}

	entries := make([]*audit.Entry, len(b.entries))
	copy(entries, b.entries)
	b.entries = make([]*audit.Entry, 0, b.limit)

	return entries
}

func (b *Buffer) RecordFailure() {
	count := b.failureCount.Inc()
	b.lastFailure.Store(time.Now().Unix())

	if count >= 3 && CircuitState(b.circuitState.Load()) == CircuitClosed {
		b.mu.Lock()
		if CircuitState(b.circuitState.Load()) == CircuitClosed {
			b.circuitState.Store(int32(CircuitOpen))
		}
		b.mu.Unlock()
	}
}

func (b *Buffer) ResetFailures() {
	b.failureCount.Store(0)
	b.circuitState.Store(int32(CircuitClosed))
}

func (b *Buffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries) >= b.limit
}

func (b *Buffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries)
}

func (b *Buffer) GetState() CircuitState {
	return CircuitState(b.circuitState.Load())
}

func (b *Buffer) SetCooldownTime(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cooldownTime = d
}
