package sim

import (
	"crypto/x509"
	"math/rand/v2"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
)

type trenovaConfig struct {
	CertificatePEM string `json:"certificatePem,omitempty"`
	InboundURL     string `json:"inboundUrl,omitempty"`
	AS2ID          string `json:"as2Id,omitempty"`
	certificate    *x509.Certificate
}

type scriptEntry struct {
	Key      string         `json:"key,omitempty"`
	Silent   bool           `json:"silent,omitempty"`
	Scenario reply.Scenario `json:"scenario"`
}

type peer struct {
	def       *partner.Definition
	guides    *oracle.GuideSet
	identity  *identity.Identity
	carrier   *reply.Carrier
	controls  partner.Controls
	epoch     atomic.Uint64
	mu        sync.Mutex
	trenova   trenovaConfig
	script    []scriptEntry
	overrides []AckOverride
	rng       *rand.Rand
	originals *originals
}

func (p *peer) name() string {
	return p.def.Name()
}

func (p *peer) trenovaSettings() trenovaConfig {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.trenova
}

func (p *peer) silent() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.def.Partner.Acknowledgment.Silent(p.rng)
}

func (p *peer) takeScript(key string) (scriptEntry, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := slices.IndexFunc(p.script, func(e scriptEntry) bool { return e.Key == key })
	if index < 0 {
		index = slices.IndexFunc(p.script, func(e scriptEntry) bool { return e.Key == "" })
	}
	if index < 0 {
		return scriptEntry{}, false
	}
	entry := p.script[index]
	p.script = slices.Delete(p.script, index, index+1)
	return entry, true
}

type original struct {
	Key string `json:"key"`
	Raw string `json:"raw"`
}

type originals struct {
	mu       sync.Mutex
	capacity int
	order    []string
	byKey    map[string]string
}

func newOriginals(capacity int) *originals {
	return &originals{capacity: max(capacity, 1), byKey: map[string]string{}}
}

func originalKey(set string, seed uint64) string {
	return set + "/" + strconv.FormatUint(seed, 10)
}

func (o *originals) remember(key, raw string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.byKey[key]; !ok {
		o.order = append(o.order, key)
	}
	o.byKey[key] = raw
	for len(o.order) > o.capacity {
		delete(o.byKey, o.order[0])
		o.order = o.order[1:]
	}
}

func (o *originals) recall(key string) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	raw, ok := o.byKey[key]
	return raw, ok
}

func (o *originals) list() []original {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]original, 0, len(o.order))
	for _, key := range o.order {
		out = append(out, original{Key: key, Raw: o.byKey[key]})
	}
	return out
}

func (o *originals) reset(list []original) {
	o.mu.Lock()
	o.order, o.byKey = nil, map[string]string{}
	o.mu.Unlock()
	for _, item := range list {
		o.remember(item.Key, item.Raw)
	}
}
