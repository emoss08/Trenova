package index

import (
	"sync"

	"github.com/emoss08/assay/internal/cache"
)

type Loader struct {
	store   *cache.Store
	fp      *Fingerprinter
	mu      sync.Mutex
	records map[string]*Record
}

func NewLoader(store *cache.Store, fp *Fingerprinter) *Loader {
	return &Loader{store: store, fp: fp, records: make(map[string]*Record)}
}

func (l *Loader) Record(importPath string) (*Record, bool) {
	if l == nil || l.store == nil || l.fp == nil {
		return nil, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if record, cached := l.records[importPath]; cached {
		return record, record != nil
	}

	payload, hit := l.store.Get(l.fp.For(importPath))
	if !hit {
		l.records[importPath] = nil

		return nil, false
	}

	record, err := Decode(payload)
	if err != nil {
		l.records[importPath] = nil

		return nil, false
	}

	l.records[importPath] = record

	return record, true
}
