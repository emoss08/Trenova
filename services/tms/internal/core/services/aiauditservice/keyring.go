package aiauditservice

import (
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/zap"
)

// Keyring holds the chain keys from configuration: the active one new rows
// are signed with, and every older one still named by rows it signed. Without
// any key the trail is still written, as an unsigned SHA-256 chain, and the
// process says so once.
type Keyring struct {
	active *aiaudit.ChainKey
	keys   map[string]*aiaudit.ChainKey
	l      *zap.Logger
	warn   sync.Once
}

func NewKeyring(cfg *config.AIAuditChainConfig, logger *zap.Logger) *Keyring {
	ring := &Keyring{
		keys: make(map[string]*aiaudit.ChainKey),
		l:    logger.Named("aiaudit.keyring"),
	}
	if cfg == nil {
		return ring
	}

	for _, key := range cfg.Keys {
		ring.keys[key.ID] = &aiaudit.ChainKey{ID: key.ID, Secret: []byte(key.Secret)}
	}
	if active, ok := cfg.ActiveKey(); ok {
		ring.active = ring.keys[active.ID]
	}

	return ring
}

// Signed reports whether new rows are signed with a key.
func (k *Keyring) Signed() bool {
	return k.active != nil
}

func (k *Keyring) ActiveKeyID() string {
	if k.active == nil {
		return ""
	}

	return k.active.ID
}

// Key is a configured key by id.
func (k *Keyring) Key(id string) (*aiaudit.ChainKey, bool) {
	key, ok := k.keys[id]

	return key, ok
}

// Signer seals rows with the active key, or as unsigned links when none is
// configured.
func (k *Keyring) Signer() repositories.AIAuditSigner {
	active := k.active
	if active == nil {
		k.warn.Do(func() {
			k.l.Warn("no AI audit chain key is configured; the trail is written as an " +
				"unsigned SHA-256 chain. Set aiAudit.chain.keys and aiAudit.chain.activeKeyId " +
				"so its rows are signed")
		})
	}

	return func(event *aiaudit.AIAuditEvent, prevHash string) error {
		return aiaudit.Seal(event, prevHash, active)
	}
}

// keyFor is the key a stored row needs to be checked, and whether that key
// is missing from configuration.
func (k *Keyring) keyFor(event *aiaudit.AIAuditEvent) (*aiaudit.ChainKey, bool) {
	if event.HashVersion != aiaudit.HashVersionSigned {
		return nil, false
	}

	key, ok := k.keys[event.HashKeyID]

	return key, !ok
}
