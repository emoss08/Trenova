package postgres

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScopeKey(size int) []byte {
	key := make([]byte, size)
	for i := range key {
		key[i] = byte(i * 7)
	}
	return key
}

func TestScopeSigner_TokenVerifiesWithDerivedPads(t *testing.T) {
	t.Parallel()

	for _, size := range []int{32, 64, 96} {
		signer, err := newScopeSigner("k1", testScopeKey(size), time.Minute)
		require.NoError(t, err)
		signer.now = func() time.Time { return time.Unix(1_700_000_000, 0) }

		tenant := dbscope.Tenant{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			UserID:         pulid.MustNew("usr_"),
		}
		token, err := signer.token(tenant)
		require.NoError(t, err)

		parts := strings.Split(token, ".")
		require.Len(t, parts, 7)
		assert.Equal(t, []string{"v1", "k1", tenant.OrganizationID.String(), tenant.BusinessUnitID.String(), tenant.UserID.String(), "1700000060"}, parts[:6])

		pads := deriveScopePads(signer.key)
		inner := sha256.Sum256(append(append([]byte{}, pads.Inner...), []byte(strings.Join(parts[:6], "."))...))
		outer := sha256.Sum256(append(append([]byte{}, pads.Outer...), inner[:]...))
		assert.Equal(t, hex.EncodeToString(outer[:]), parts[6], "key size %d", size)
	}
}

func TestScopeSigner_OmitsMissingUser(t *testing.T) {
	t.Parallel()

	signer, err := newScopeSigner("k1", testScopeKey(32), time.Minute)
	require.NoError(t, err)

	token, err := signer.token(dbscope.Tenant{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.NoError(t, err)
	assert.Equal(t, "-", strings.Split(token, ".")[4])
}

func TestScopeSigner_RejectsIncompleteOrMalformedTenants(t *testing.T) {
	t.Parallel()

	signer, err := newScopeSigner("k1", testScopeKey(32), time.Minute)
	require.NoError(t, err)

	_, err = signer.token(dbscope.Tenant{OrganizationID: pulid.MustNew("org_")})
	require.ErrorIs(t, err, ErrInvalidTenantScope)

	_, err = signer.token(dbscope.Tenant{
		OrganizationID: pulid.ID("org_x'; RESET trenova.scope; --"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.ErrorIs(t, err, ErrMalformedTenantScope)

	_, err = signer.token(dbscope.Tenant{
		OrganizationID: pulid.ID("org.a"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	require.ErrorIs(t, err, ErrMalformedTenantScope)
}

func TestNewScopeSigner_RejectsWeakConfiguration(t *testing.T) {
	t.Parallel()

	_, err := newScopeSigner("k1", testScopeKey(16), time.Minute)
	require.ErrorIs(t, err, ErrInvalidScopeKey)

	_, err = newScopeSigner("bad id", testScopeKey(32), time.Minute)
	require.ErrorIs(t, err, ErrInvalidScopeKey)

	_, err = newScopeSigner("k1", testScopeKey(32), 0)
	require.ErrorIs(t, err, ErrInvalidScopeKey)
}

func TestBeginCommand(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		isolation int
		readOnly  bool
		want      string
	}{
		"default":             {want: "BEGIN"},
		"read only":           {readOnly: true, want: "BEGIN READ ONLY"},
		"serializable":        {isolation: 6, want: "BEGIN ISOLATION LEVEL SERIALIZABLE"},
		"repeatable readonly": {isolation: 4, readOnly: true, want: "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY"},
	}
	for name, tc := range cases {
		got, err := beginCommand(driverTxOptions(tc.isolation, tc.readOnly))
		require.NoError(t, err, name)
		assert.Equal(t, tc.want, got, name)
	}

	_, err := beginCommand(driverTxOptions(7, false))
	require.Error(t, err)
}

func TestScramSHA256Verifier_MatchesPostgresFormat(t *testing.T) {
	t.Parallel()

	verifier, err := scramSHA256VerifierWithSalt("pencil", []byte("0123456789abcdef"))
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(verifier, "SCRAM-SHA-256$4096:MDEyMzQ1Njc4OWFiY2RlZg==$"))
	assert.Len(t, strings.Split(strings.Split(verifier, "$")[2], ":"), 2)
}

func driverTxOptions(isolation int, readOnly bool) driver.TxOptions {
	return driver.TxOptions{Isolation: driver.IsolationLevel(isolation), ReadOnly: readOnly}
}
