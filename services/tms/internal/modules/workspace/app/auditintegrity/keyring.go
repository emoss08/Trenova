package auditintegrity

import (
	"crypto/hmac"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
)

type Keyring struct {
	activeID string
	keys     map[string][]byte
}

func NewKeyring(cfg *aiauditconfig.Config) *Keyring {
	ring := &Keyring{keys: make(map[string][]byte)}
	if cfg == nil {
		return ring
	}
	for _, key := range cfg.Chain.Keys {
		ring.keys[key.ID] = []byte(key.Secret)
	}
	if active, ok := cfg.Chain.ActiveKey(); ok {
		ring.activeID = active.ID
	}
	return ring
}

func (k *Keyring) Signed() bool {
	return k != nil && k.activeID != ""
}

func (k *Keyring) Sign(seal *audit.Seal) {
	if !k.Signed() {
		seal.Sign("", nil)
		return
	}
	seal.Sign(k.activeID, k.keys[k.activeID])
}

func (k *Keyring) Check(seal *audit.Seal) (ok, keyMissing bool) {
	var secret []byte
	if seal.HashVersion != audit.SealHashVersionUnsigned {
		var found bool
		if secret, found = k.keys[seal.HashKeyID]; !found {
			return false, true
		}
	}
	return hmac.Equal([]byte(seal.ComputeHash(secret)), []byte(seal.Hash)), false
}
