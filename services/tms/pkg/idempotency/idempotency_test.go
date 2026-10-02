package idempotency_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/idempotency"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func testScope() idempotency.Scope {
	return idempotency.Scope{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		PrincipalType:  "user",
		PrincipalID:    pulid.MustNew("usr_"),
	}
}

func TestScopedKeyIsStableAndScoped(t *testing.T) {
	hasher := idempotency.NewHasher("secret")
	scope := testScope()
	other := scope
	other.PrincipalID = pulid.MustNew("usr_")

	key := hasher.ScopedKey(scope, "k")
	assert.Len(t, key, idempotency.ScopedKeyLen)
	assert.Equal(t, key, hasher.ScopedKey(scope, "k"))
	assert.NotEqual(t, key, hasher.ScopedKey(other, "k"))
	assert.NotEqual(t, key, hasher.ScopedKey(scope, "k2"))
}

func TestScopedKeyDependsOnTheSecret(t *testing.T) {
	scope := testScope()

	assert.NotEqual(t,
		idempotency.NewHasher("one").ScopedKey(scope, "k"),
		idempotency.NewHasher("two").ScopedKey(scope, "k"),
	)
}

func TestFingerprintSeparatesItsParts(t *testing.T) {
	hasher := idempotency.NewHasher("secret")

	base := hasher.Fingerprint("POST", "/a", []byte("b"))
	assert.Equal(t, base, hasher.Fingerprint("POST", "/a", []byte("b")))
	assert.NotEqual(t, base, hasher.Fingerprint("POST", "/ab", nil))
	assert.NotEqual(t, base, hasher.Fingerprint("PUT", "/a", []byte("b")))
}

func TestValidClientKey(t *testing.T) {
	assert.True(t, idempotency.ValidClientKey("0b6f3c0e-1f6a-4d0c-9c8e-1d2a3b4c5d6e"))
	assert.False(t, idempotency.ValidClientKey(""))
	assert.False(t, idempotency.ValidClientKey("has space"))
	assert.False(t, idempotency.ValidClientKey("tab\tkey"))
	assert.False(t, idempotency.ValidClientKey(strings.Repeat("k", idempotency.MaxKeyLength+1)))
}
