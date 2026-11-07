package auditintegrity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
)

const sealDomain = "trenova.audit-seal/v1"

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
	if k.Signed() {
		seal.HashKeyID = k.activeID
		seal.HashVersion = audit.SealHashVersionSigned
	} else {
		seal.HashKeyID = ""
		seal.HashVersion = audit.SealHashVersionUnsigned
	}
	seal.Hash, _ = k.hash(seal)
}

func (k *Keyring) Check(seal *audit.Seal) (ok, keyMissing bool) {
	expected, found := k.hash(seal)
	if !found {
		return false, true
	}
	return hmac.Equal([]byte(expected), []byte(seal.Hash)), false
}

func (k *Keyring) hash(seal *audit.Seal) (string, bool) {
	message := sealMessage(seal)
	if seal.HashVersion == audit.SealHashVersionUnsigned {
		sum := sha256.Sum256([]byte(message))
		return hex.EncodeToString(sum[:]), true
	}
	secret, ok := k.keys[seal.HashKeyID]
	if !ok {
		return "", false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil)), true
}

func sealMessage(seal *audit.Seal) string {
	classes := make([]string, 0, len(seal.ClassDigests))
	for class := range seal.ClassDigests {
		classes = append(classes, class)
	}
	slices.Sort(classes)

	var b strings.Builder
	b.WriteString(sealDomain)
	for _, part := range []string{
		strconv.Itoa(int(seal.HashVersion)),
		seal.HashKeyID,
		seal.PrevHash,
		seal.OrganizationID.String(),
		seal.BusinessUnitID.String(),
		strconv.FormatInt(seal.Seq, 10),
		strconv.FormatInt(seal.FromTS, 10),
		strconv.FormatInt(seal.ToTS, 10),
		strconv.FormatInt(seal.RowCount, 10),
	} {
		b.WriteByte('\n')
		b.WriteString(part)
	}
	for _, class := range classes {
		digest := seal.ClassDigests[class]
		b.WriteByte('\n')
		b.WriteString(class)
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(digest.Count, 10))
		b.WriteByte(':')
		b.WriteString(digest.Digest)
	}
	return b.String()
}

