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
	circuitState atomic.Int32 // Using atomic for faster state checks without locking
	lastFailure  atomic.Int64 // Store as Unix timestamp for atomic access
	cooldownTime time.Duration
}

func NewBuffer(limit int) *Buffer {
	return &Buffer{
		entries:      make([]*audit.Entry, 0, limit),
		limit:        limit,
		failureCount: atomic.NewInt64(0),
		circuitState: *atomic.NewInt32(int32(CircuitClosed)),
		lastFailure:  *atomic.NewInt64(0),
		cooldownTime: 30 * time.Second, // Default cooldown time
	}
}

// Add adds an audit entry to the buffer.
// Returns true if the entry was added, false if rejected due to circuit breaker
func (b *Buffer) Add(entry *audit.Entry) bool {
	// Fast path: check circuit state without lock
	currentState := CircuitState(b.circuitState.Load())
	if currentState == CircuitOpen {
		// Use atomic operations for timestamp comparison to avoid lock
		lastFailureTime := time.Unix(b.lastFailure.Load(), 0)
		if time.Since(lastFailureTime) > b.cooldownTime {
			// Try to transition to half-open
			b.circuitState.CompareAndSwap(int32(CircuitOpen), int32(CircuitHalfOpen))
		} else {
			return false // Reject entry while circuit is open
		}
	}

	// Only lock for the actual buffer manipulation
	b.mu.Lock()
	defer b.mu.Unlock()

	// Final capacity check under lock
	if len(b.entries) >= b.limit {
		return false
	}

	// Add entry to buffer - use append with pre-allocated slice when possible
	b.entries = append(b.entries, entry)

	// If we're in half-open state and successfully added, reset to closed
	if CircuitState(b.circuitState.Load()) == CircuitHalfOpen {
		b.circuitState.Store(int32(CircuitClosed))
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

	// Create new slice with exact capacity needed
	entries := make([]*audit.Entry, len(b.entries))
	copy(entries, b.entries)

	// Reset the buffer more efficiently by creating a new slice
	// This allows the old one to be garbage collected if no longer referenced
	b.entries = make([]*audit.Entry, 0, b.limit)

	return entries
}

// RecordFailure records a failure and potentially opens the circuit
func (b *Buffer) RecordFailure() {
	count := b.failureCount.Inc()
	b.lastFailure.Store(time.Now().Unix())

	// If we've had too many consecutive failures, open the circuit
	// Only take a lock if we're actually going to change state
	if count >= 3 && CircuitState(b.circuitState.Load()) == CircuitClosed {
		b.mu.Lock()
		// Double-check state after acquiring lock
		if CircuitState(b.circuitState.Load()) == CircuitClosed {
			b.circuitState.Store(int32(CircuitOpen))
		}
		b.mu.Unlock()
	}
}

// ResetFailures resets the failure count and closes the circuit
func (b *Buffer) ResetFailures() {
	b.failureCount.Store(0)
	b.circuitState.Store(int32(CircuitClosed))
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
	return CircuitState(b.circuitState.Load())
}

// SetCooldownTime sets the cooldown time for the circuit breaker
func (b *Buffer) SetCooldownTime(d time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cooldownTime = d
}
