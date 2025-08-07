/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package redis

import (
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rotisserie/eris"
)

var (
	// Nil reply returned by Redis when key does not exist.
	ErrNil = redis.Nil
	// Error returned when the redis config is nil
	ErrConfigNil = eris.New("redis config is nil")

	// Error returned when the redis circuit breaker is open
	ErrCircuitBreakerOpen = eris.New("redis circuit breaker is open")
)

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int32

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements a simple circuit breaker pattern for Redis operations
type CircuitBreaker struct {
	state        int32 // atomic access - CircuitBreakerState
	failures     int32 // atomic access
	lastFailTime int64 // atomic access - unix timestamp
	threshold    int32
	timeout      time.Duration
	resetTimeout time.Duration
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int32, timeout, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        int32(CircuitBreakerClosed),
		threshold:    threshold,
		timeout:      timeout,
		resetTimeout: resetTimeout,
	}
}

// CanExecute checks if an operation can be executed
func (cb *CircuitBreaker) CanExecute() bool {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))

	switch state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		lastFailTime := atomic.LoadInt64(&cb.lastFailTime)
		if time.Since(time.Unix(lastFailTime, 0)) > cb.resetTimeout {
			// Try to transition to half-open
			if atomic.CompareAndSwapInt32(
				&cb.state,
				int32(CircuitBreakerOpen),
				int32(CircuitBreakerHalfOpen),
			) {
				atomic.StoreInt32(&cb.failures, 0)
				return true
			}
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// OnSuccess records a successful operation
func (cb *CircuitBreaker) OnSuccess() {
	state := CircuitBreakerState(atomic.LoadInt32(&cb.state))
	if state == CircuitBreakerHalfOpen {
		atomic.StoreInt32(&cb.state, int32(CircuitBreakerClosed))
		atomic.StoreInt32(&cb.failures, 0)
	}
}

// OnFailure records a failed operation
func (cb *CircuitBreaker) OnFailure() {
	failures := atomic.AddInt32(&cb.failures, 1)
	atomic.StoreInt64(&cb.lastFailTime, time.Now().Unix())

	if failures >= cb.threshold {
		atomic.StoreInt32(&cb.state, int32(CircuitBreakerOpen))
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}
