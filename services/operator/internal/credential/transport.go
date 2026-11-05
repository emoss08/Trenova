package credential

import (
	"errors"
	"net/http"
	"sync/atomic"
	"time"
)

var ErrNoIdentity = errors.New("no identity is installed")

type Transport struct {
	current atomic.Pointer[http.Transport]
}

func (t *Transport) Use(ident *Identity) error {
	cfg, err := ident.TLSConfig()
	if err != nil {
		return err
	}
	next := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSClientConfig:       cfg,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          4,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	if prev := t.current.Swap(next); prev != nil {
		prev.CloseIdleConnections()
	}
	return nil
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	cur := t.current.Load()
	if cur == nil {
		return nil, ErrNoIdentity
	}
	return cur.RoundTrip(req)
}
