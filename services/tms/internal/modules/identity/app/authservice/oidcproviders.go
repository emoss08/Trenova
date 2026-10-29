package authservice

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sso"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
)

const oidcRequestTimeout = 15 * time.Second

type oidcDiscoverFunc func(ctx context.Context, issuer string) (*oidc.Provider, error)

type oidcProviderEntry struct {
	fingerprint string
	provider    *oidc.Provider
	expiresAt   time.Time
}

type oidcProviders struct {
	mu       sync.Mutex
	entries  map[pulid.ID]oidcProviderEntry
	client   *http.Client
	policy   httpsafe.Policy
	ttl      time.Duration
	discover oidcDiscoverFunc
	now      func() time.Time
}

func newOIDCProviders(policy httpsafe.Policy, ttl time.Duration) *oidcProviders {
	return &oidcProviders{
		entries:  make(map[pulid.ID]oidcProviderEntry),
		client:   httpsafe.NewClientWithPolicy(oidcRequestTimeout, policy),
		policy:   policy,
		ttl:      ttl,
		discover: oidc.NewProvider,
		now:      time.Now,
	}
}

func (p *oidcProviders) clientContext(ctx context.Context) context.Context {
	return oidc.ClientContext(ctx, p.client)
}

func (p *oidcProviders) get(
	ctx context.Context,
	cfg *sso.SSOConfig,
) (*oidc.Provider, error) {
	if _, err := httpsafe.ValidateHTTPSURL(cfg.OIDCIssuerURL, p.policy); err != nil {
		return nil, err
	}

	fingerprint := oidcConfigFingerprint(cfg)
	now := p.now()

	p.mu.Lock()
	entry, ok := p.entries[cfg.ID]
	p.mu.Unlock()
	if ok && entry.fingerprint == fingerprint && now.Before(entry.expiresAt) {
		return entry.provider, nil
	}

	provider, err := p.discover(p.clientContext(ctx), cfg.OIDCIssuerURL)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	for id, existing := range p.entries {
		if !now.Before(existing.expiresAt) {
			delete(p.entries, id)
		}
	}
	p.entries[cfg.ID] = oidcProviderEntry{
		fingerprint: fingerprint,
		provider:    provider,
		expiresAt:   now.Add(p.ttl),
	}
	return provider, nil
}

func oidcConfigFingerprint(cfg *sso.SSOConfig) string {
	return cfg.OIDCIssuerURL + "\x00" +
		strconv.FormatInt(cfg.Version, 10) + "\x00" +
		strconv.FormatInt(cfg.UpdatedAt, 10)
}
