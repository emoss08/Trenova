package nsconnector

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

type accountSlots struct {
	slots chan struct{}
	users int
}

type accountGate struct {
	limit     int
	mu        sync.Mutex
	byAccount map[string]*accountSlots
}

func newAccountGate() *accountGate {
	return &accountGate{
		limit:     maxConcurrentPerAccount,
		byAccount: make(map[string]*accountSlots),
	}
}

func (g *accountGate) transport(next http.RoundTripper) http.RoundTripper {
	return &gatedTransport{gate: g, next: next}
}

func (g *accountGate) join(account string) *accountSlots {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry, ok := g.byAccount[account]
	if !ok {
		entry = &accountSlots{slots: make(chan struct{}, g.limit)}
		g.byAccount[account] = entry
	}
	entry.users++
	return entry
}

func (g *accountGate) leave(account string, entry *accountSlots) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry.users--
	if entry.users == 0 && g.byAccount[account] == entry {
		delete(g.byAccount, account)
	}
}

func accountKey(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	return strings.ToLower(req.URL.Host)
}

type gatedTransport struct {
	gate *accountGate
	next http.RoundTripper
}

func (t *gatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	account := accountKey(req)
	if account == "" {
		return t.next.RoundTrip(req)
	}

	entry := t.gate.join(account)
	select {
	case entry.slots <- struct{}{}:
	case <-req.Context().Done():
		t.gate.leave(account, entry)
		return nil, req.Context().Err()
	}

	var once sync.Once
	release := func() {
		once.Do(func() {
			<-entry.slots
			t.gate.leave(account, entry)
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
