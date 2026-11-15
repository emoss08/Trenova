package supportaccess_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
)

const (
	testInstance = "inst_01JTESTINSTANCE"
	testDomain   = "trenova.app"
	testKeyID    = "cp-2026-11"
	testOrg      = "org_01JA2B3C4D5E6F7G8H9JKMNPQR"
)

type signingKey struct {
	id   string
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

func newSigningKey(t *testing.T, id string) signingKey {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return signingKey{id: id, priv: priv, pub: pub}
}

func (k signingKey) jwk() supportaccess.SigningKey {
	return supportaccess.SigningKey{
		KeyID:     k.id,
		KeyType:   "OKP",
		Curve:     "Ed25519",
		Algorithm: "EdDSA",
		Use:       "sig",
		X:         base64.RawURLEncoding.EncodeToString(k.pub),
	}
}

func keySetOf(keys ...signingKey) *supportaccess.KeySet {
	set := &supportaccess.KeySet{InstanceID: testInstance, StaffDomain: testDomain}
	for _, k := range keys {
		set.Keys = append(set.Keys, k.jwk())
	}
	return set
}

func claimsAt(now time.Time) map[string]any {
	return map[string]any{
		"iss":  supportaccess.AssertionIssuer,
		"sub":  "jane.doe@trenova.app",
		"aud":  testInstance,
		"jti":  "a1b2c3d4e5f6g7h8i9j0",
		"iat":  now.Unix(),
		"exp":  now.Add(time.Minute).Unix(),
		"name": "Jane Doe",
		"org":  testOrg,
	}
}

func sign(t *testing.T, key signingKey, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{Key: key.priv, KeyID: key.id}},
		(&jose.SignerOptions{}).WithType(supportaccess.AssertionType),
	)
	require.NoError(t, err)
	return serialize(t, signer, claims)
}

func serialize(t *testing.T, signer jose.Signer, claims map[string]any) string {
	t.Helper()
	payload, err := sonic.Marshal(claims)
	require.NoError(t, err)
	obj, err := signer.Sign(payload)
	require.NoError(t, err)
	token, err := obj.CompactSerialize()
	require.NoError(t, err)
	return token
}
