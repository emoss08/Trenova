package xeroconnector

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

const tenantHeader = "xero-tenant-id"

type tenantSlots struct {
	slots chan struct{}
	users int
}

type tenantGate struct {
	limit int
	mu    sync.Mutex
	byOrg map[string]*tenantSlots
}

func newTenantGate() *tenantGate {
	return &tenantGate{limit: maxConcurrentPerOrg, byOrg: make(map[string]*tenantSlots)}
}

func (g *tenantGate) transport(next http.RoundTripper) http.RoundTripper {
	return &gatedTransport{gate: g, next: next}
}

func (g *tenantGate) join(tenant string) *tenantSlots {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.byOrg[tenant]
	if !ok {
		entry = &tenantSlots{slots: make(chan struct{}, g.limit)}
		g.byOrg[tenant] = entry
	}
	entry.users++
	return entry
}

func (g *tenantGate) leave(tenant string, entry *tenantSlots) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry.users--
	if entry.users == 0 && g.byOrg[tenant] == entry {
		delete(g.byOrg, tenant)
	}
}

type gatedTransport struct {
	gate *tenantGate
	next http.RoundTripper
}

func (t *gatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tenant := strings.TrimSpace(req.Header.Get(tenantHeader))
	if tenant == "" {
		return t.next.RoundTrip(req)
	}

	entry := t.gate.join(tenant)
	select {
	case entry.slots <- struct{}{}:
	case <-req.Context().Done():
		t.gate.leave(tenant, entry)
		return nil, req.Context().Err()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			<-entry.slots
			t.gate.leave(tenant, entry)
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
