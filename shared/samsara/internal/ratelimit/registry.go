package ratelimit

import (
	"crypto/sha256"
	"sync"
	"time"
)

const defaultIdleTTL = time.Hour

type Registry struct {
	mu        sync.Mutex
	sets      map[[sha256.Size]byte]*Set
	idleTTL   time.Duration
	lastSweep time.Time
	now       func() time.Time
}

var shared = NewRegistry(defaultIdleTTL)

func NewRegistry(idleTTL time.Duration) *Registry {
	return newRegistry(idleTTL, time.Now)
}

func newRegistry(idleTTL time.Duration, now func() time.Time) *Registry {
	if idleTTL <= 0 {
		idleTTL = defaultIdleTTL
	}
	return &Registry{
		sets:      make(map[[sha256.Size]byte]*Set),
		idleTTL:   idleTTL,
		lastSweep: now(),
		now:       now,
	}
}

func Shared(baseURL, token string) *Set {
	return shared.For(baseURL, token)
}

func (r *Registry) For(baseURL, token string) *Set {
	key := credentialKey(baseURL, token)
	now := r.now()

	r.mu.Lock()
	defer r.mu.Unlock()

	if now.Sub(r.lastSweep) >= r.idleTTL {
		r.sweepLocked(now)
	}

	if set, ok := r.sets[key]; ok {
		set.touch()
		return set
	}
	set := newSet(r.now)
	r.sets[key] = set
	return set
}

func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sets)
}

func (r *Registry) sweepLocked(now time.Time) {
	for key, set := range r.sets {
		if set.idleSince(now) >= r.idleTTL {
			delete(r.sets, key)
		}
	}
	r.lastSweep = now
}

func credentialKey(baseURL, token string) [sha256.Size]byte {
	h := sha256.New()
	h.Write([]byte(baseURL))
	h.Write([]byte{0})
	h.Write([]byte(token))
	var key [sha256.Size]byte
	h.Sum(key[:0])
	return key
}
