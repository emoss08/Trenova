package bcconnector

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
)

const companiesSegment = "/companies("

type companySlots struct {
	slots chan struct{}
	users int
}

type companyGate struct {
	limit     int
	mu        sync.Mutex
	byCompany map[string]*companySlots
}

func newCompanyGate() *companyGate {
	return &companyGate{
		limit:     maxConcurrentPerCompany,
		byCompany: make(map[string]*companySlots),
	}
}

func (g *companyGate) transport(next http.RoundTripper) http.RoundTripper {
	return &gatedTransport{gate: g, next: next}
}

func (g *companyGate) join(company string) *companySlots {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.byCompany[company]
	if !ok {
		entry = &companySlots{slots: make(chan struct{}, g.limit)}
		g.byCompany[company] = entry
	}
	entry.users++
	return entry
}

func (g *companyGate) leave(company string, entry *companySlots) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry.users--
	if entry.users == 0 && g.byCompany[company] == entry {
		delete(g.byCompany, company)
	}
}

func companyKey(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	path := req.URL.Path
	start := strings.Index(path, companiesSegment)
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(path[start:], ')')
	if end < 0 {
		return ""
	}
	return strings.ToLower(req.URL.Host + path[:start+end+1])
}

type gatedTransport struct {
	gate *companyGate
	next http.RoundTripper
}

func (t *gatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	company := companyKey(req)
	if company == "" {
		return t.next.RoundTrip(req)
	}

	entry := t.gate.join(company)
	select {
	case entry.slots <- struct{}{}:
	case <-req.Context().Done():
		t.gate.leave(company, entry)
		return nil, req.Context().Err()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			<-entry.slots
			t.gate.leave(company, entry)
		})
	}

	resp, err := t.next.RoundTrip(req)
	if err != nil || resp == nil || resp.Body == nil {
		release()
		return resp, err
	}
	resp.Body = &releasingBody{ReadCloser: resp.Body, release: release}
	return resp, nil
}

type releasingBody struct {
	io.ReadCloser
	release func()
}

func (b *releasingBody) Close() error {
	err := b.ReadCloser.Close()
	b.release()
	return err
}

type batchLock struct {
	held  chan struct{}
	users int
}

type batchLocks struct {
	mu      sync.Mutex
	byBatch map[string]*batchLock
}

func newBatchLocks() *batchLocks {
	return &batchLocks{byBatch: make(map[string]*batchLock)}
}

func (l *batchLocks) acquire(ctx context.Context, key string) (func(), error) {
	l.mu.Lock()
	entry, ok := l.byBatch[key]
	if !ok {
		entry = &batchLock{held: make(chan struct{}, 1)}
		l.byBatch[key] = entry
	}
	entry.users++
	l.mu.Unlock()

	select {
	case entry.held <- struct{}{}:
	case <-ctx.Done():
		l.forget(key, entry)
		return nil, ctx.Err()
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			<-entry.held
			l.forget(key, entry)
		})
	}, nil
}

func (l *batchLocks) forget(key string, entry *batchLock) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry.users--
	if entry.users == 0 && l.byBatch[key] == entry {
		delete(l.byBatch, key)
	}
}
