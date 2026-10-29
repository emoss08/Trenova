package bcconnector

import (
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
