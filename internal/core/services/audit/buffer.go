package audit

import (
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"go.uber.org/atomic"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // Normal operation
	CircuitOpen                         // Rejecting entries due to backpressure
	CircuitHalfOpen                     // Testing if system can accept entries again
)

// Buffer is a thread-safe buffer for audit entries with circuit breaker capability
type Buffer struct {
	entries      []*audit.Entry
	mu           sync.RWMutex
	limit        int
	failureCount *atomic.Int64
	circuitState CircuitState
	lastFailure  time.Time
	cooldownTime time.Duration
}

func NewBuffer(limit int) *Buffer {
	return &Buffer{
		entries:      make([]*audit.Entry, 0, limit),
		limit:        limit,
		failureCount: atomic.NewInt64(0),
		circuitState: CircuitClosed,
		cooldownTime: 30 * time.Second, // Default cooldown time
	}
}

// Add adds an audit entry to the buffer.
// Returns true if the entry was added, false if rejected due to circuit breaker
func (b *Buffer) Add(entry *audit.Entry) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Check circuit breaker state
	if b.circuitState == CircuitOpen {
		// Check if cooldown period has elapsed
		if time.Since(b.lastFailure) > b.cooldownTime {
			b.circuitState = CircuitHalfOpen
		} else {
			return false // Reject entry while circuit is open
		}
	}

	// Add entry to buffer
	b.entries = append(b.entries, entry)

	// If we're in half-open state and successfully added, reset to closed
	if b.circuitState == CircuitHalfOpen {
		b.circuitState = CircuitClosed
		b.failureCount.Store(0)
	}

	return true
}

// FlushAndReset returns the current entries in the buffer and resets the buffer.
func (b *Buffer) FlushAndReset() []*audit.Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		return nil
	}

	entries := make([]*audit.Entry, len(b.entries))
	copy(entries, b.entries)

	// Reset the buffer
	b.entries = make([]*audit.Entry, 0, b.limit)

	return entries
}

// RecordFailure records a failure and potentially opens the circuit
func (b *Buffer) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	count := b.failureCount.Inc()
	b.lastFailure = time.Now()

	// If we've had too many consecutive failures, open the circuit
	if count >= 3 && b.circuitState == CircuitClosed {
		b.circuitState = CircuitOpen
	}
}

// ResetFailures resets the failure count and closes the circuit
func (b *Buffer) ResetFailures() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.failureCount.Store(0)
	b.circuitState = CircuitClosed
}

// IsFull returns true if the buffer is full.
func (b *Buffer) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries) >= b.limit
}

// Size returns the number of entries in the buffer.
func (b *Buffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.entries)
}

// GetState returns the current circuit state
func (b *Buffer) GetState() CircuitState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.circuitState
}

// SetCooldownTime sets the cooldown time for the circuit breaker
func (b *Buffer) SetCooldownTime(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cooldownTime = d
}
