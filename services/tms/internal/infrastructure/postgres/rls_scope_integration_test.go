//go:build integration

package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	refusedNoScope    = "no tenant scope is set for this transaction"
	refusedMalformed  = "tenant scope is malformed"
	refusedUnknownKey = "tenant scope was signed with an unknown or retired key"
	refusedSignature  = "tenant scope signature does not verify"
	refusedExpired    = "tenant scope has expired"
	insufficientPriv  = "42501"
)

func (f *rlsFixture) appSession(t *testing.T) *pgx.Conn {
	t.Helper()

	conn, err := pgx.Connect(
		t.Context(),
		f.cfg.Database.DSN(f.cfg.App.Name, f.cfg.Database.Password),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(context.Background()) })

	return conn
}

func (f *rlsFixture) signedToken(
	t *testing.T,
	keyID string,
	key []byte,
	tenant dbscope.Tenant,
) string {
	t.Helper()

	signer, err := newScopeSigner(keyID, key, f.cfg.Database.RLS.GetScopeTTL())
	require.NoError(t, err)
	token, err := signer.token(tenant)
	require.NoError(t, err)

	return token
}

func signScopePayload(key []byte, payload string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(payload))

	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func scopePayload(keyID string, tenant dbscope.Tenant, expires string) string {
	return strings.Join([]string{
		scopeTokenVersion,
		keyID,
		tenant.OrganizationID.String(),
		tenant.BusinessUnitID.String(),
		scopeNoUser,
		expires,
	}, ".")
}

func fleetCodesUnder(ctx context.Context, conn *pgx.Conn, tokens ...string) ([][]string, error) {
	seen := make([][]string, 0, len(tokens))
	err := pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		for _, token := range tokens {
			if token != "" {
				if _, err := tx.Exec(ctx, setScopeForTestSQL, token); err != nil {
					return err
				}
			}
			rows, err := tx.Query(ctx, "SELECT code FROM fleet_codes ORDER BY code")
			if err != nil {
				return err
			}
			codes, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				return err
			}
			seen = append(seen, codes)
		}
		return nil
	})

	return seen, err
}

func requireRefused(t *testing.T, err error, message string) {
	t.Helper()

	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	assert.Equal(t, insufficientPriv, pgErr.Code)
	assert.Equal(t, message, pgErr.Message)
}

func TestRLS_ScopeTokenIsRefusedUnlessSignedCurrentAndWellFormed(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.appSession(t)

	valid := f.signedToken(t, "ktest", f.key, f.tenantA)
	otherKey := make([]byte, 48)
	_, err := rand.Read(otherKey)
	require.NoError(t, err)
	future := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	parts := strings.Split(valid, ".")
	require.Len(t, parts, 7)

	cases := []struct {
		name    string
		token   string
		refusal string
	}{
		{"an empty scope", "", refusedNoScope},
		{
			"another token version",
			"v2" + strings.TrimPrefix(valid, scopeTokenVersion),
			refusedMalformed,
		},
		{"a missing signature", strings.Join(parts[:6], "."), refusedMalformed},
		{"an extra segment", valid + ".x", refusedMalformed},
		{"a bare word", "garbage", refusedMalformed},
		{
			"a key that was never installed",
			f.signedToken(t, "knone", f.key, f.tenantA),
			refusedUnknownKey,
		},
		{"a forged signature", f.signedToken(t, "ktest", otherKey, f.tenantB), refusedSignature},
		{
			"another tenant under a valid signature",
			strings.Replace(
				valid,
				f.tenantA.OrganizationID.String(),
				f.tenantB.OrganizationID.String(),
				1,
			),
			refusedSignature,
		},
		{
			"an upper-case signature",
			strings.Join(parts[:6], ".") + "." + strings.ToUpper(parts[6]),
			refusedSignature,
		},
		{
			"a signed past expiry",
			signScopePayload(f.key, scopePayload("ktest", f.tenantA, "1")),
			refusedExpired,
		},
		{
			"a signed expiry that is not a number",
			signScopePayload(f.key, scopePayload("ktest", f.tenantA, "soon")),
			refusedExpired,
		},
		{
			"a signed expiry past twelve digits",
			signScopePayload(f.key, scopePayload("ktest", f.tenantA, "9"+future+"00")),
			refusedExpired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := fleetCodesUnder(t.Context(), conn, tc.token)
			requireRefused(t, err, tc.refusal)
		})
	}

	seen, err := fleetCodesUnder(t.Context(), conn, valid)
	require.NoError(t, err)
	assert.Equal(
		t,
		[][]string{{"FCA"}},
		seen,
		"the session still reads with a valid scope after every refusal",
	)
}

func TestRLS_ScopeIsVerifiedAgainOnEveryChangeAndEveryTransaction(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.appSession(t)

	tokenA := f.signedToken(t, "ktest", f.key, f.tenantA)
	tokenB := f.signedToken(t, "ktest", f.key, f.tenantB)

	seen, err := fleetCodesUnder(t.Context(), conn, tokenA, tokenB, tokenA)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"FCA"}, {"FCB"}, {"FCA"}}, seen)

	forger := make([]byte, 48)
	_, err = rand.Read(forger)
	require.NoError(t, err)
	forged := f.signedToken(t, "ktest", forger, f.tenantB)
	_, err = fleetCodesUnder(t.Context(), conn, tokenA, forged)
	requireRefused(t, err, refusedSignature)

	_, err = fleetCodesUnder(t.Context(), conn, "")
	requireRefused(t, err, refusedNoScope)

	seen, err = fleetCodesUnder(t.Context(), conn, tokenB)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"FCB"}}, seen)

	_, err = fleetCodesUnder(t.Context(), conn, "")
	requireRefused(t, err, refusedNoScope)
}

func TestRLS_RetiringAKeyRefusesATokenItSignedBefore(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.appSession(t)

	oldKey := make([]byte, 48)
	_, err := rand.Read(oldKey)
	require.NoError(t, err)
	require.NoError(t, InstallRLSScopeKey(t.Context(), f.admin, "kold", oldKey))

	token := f.signedToken(t, "kold", oldKey, f.tenantA)
	seen, err := fleetCodesUnder(t.Context(), conn, token)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"FCA"}}, seen)

	retired, err := RetireRLSScopeKey(t.Context(), f.admin, "kold")
	require.NoError(t, err)
	require.True(t, retired)

	_, err = fleetCodesUnder(t.Context(), conn, token)
	requireRefused(t, err, refusedUnknownKey)
}

func TestRLS_EveryMutationOfASignedScopeIsRefused(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.appSession(t)

	valid := f.signedToken(t, "ktest", f.key, f.tenantA)
	mutations := scopeMutations(valid)
	require.NotEmpty(t, mutations)

	refusals := map[string]bool{
		refusedNoScope:    true,
		refusedMalformed:  true,
		refusedUnknownKey: true,
		refusedSignature:  true,
		refusedExpired:    true,
	}
	for _, mutated := range mutations {
		_, err := fleetCodesUnder(t.Context(), conn, mutated)
		var pgErr *pgconn.PgError
		if !assert.ErrorAs(t, err, &pgErr, "accepted %q", mutated) {
			continue
		}
		assert.Equal(t, insufficientPriv, pgErr.Code, mutated)
		assert.True(t, refusals[pgErr.Message], "%q refused with %q", mutated, pgErr.Message)
	}
}

func scopeMutations(token string) []string {
	const alphabet = "0aZ.-_9f"

	seen := map[string]bool{token: true}
	out := make([]string, 0, len(token)*4)
	add := func(candidate string) {
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		out = append(out, candidate)
	}

	for i := range len(token) {
		for _, r := range alphabet {
			if byte(r) != token[i] {
				add(token[:i] + string(r) + token[i+1:])
				break
			}
		}
		add(token[:i] + token[i+1:])
		add(token[:i] + token[i:i+1] + token[i:])
		add(token[:i])
	}

	parts := strings.Split(token, ".")
	for i := range parts {
		for j := i + 1; j < len(parts); j++ {
			swapped := append([]string(nil), parts...)
			swapped[i], swapped[j] = swapped[j], swapped[i]
			add(strings.Join(swapped, "."))
		}
	}

	return out
}

func TestRLS_PoolsRunWithoutJIT(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)
	conn := f.connect(t)

	for name, ctx := range map[string]context.Context{
		"tenant": dbscope.WithTenant(t.Context(), f.tenantA),
		"system": dbscope.WithSystem(t.Context(), "integration test jit setting"),
	} {
		jit, err := currentSetting(ctx, conn, "jit")
		require.NoError(t, err, name)
		assert.Equal(t, "off", jit, name)
	}
}

func TestRLS_ProvisionedRolesLogInWithoutJIT(t *testing.T) {
	f := newRLSFixture(t, databaseconfig.RLSModeEnforce)

	for _, dsn := range []string{
		f.cfg.Database.DSN(f.cfg.App.Name, f.cfg.Database.Password),
		f.cfg.Database.DSNForRole(f.cfg.App.Name, f.cfg.Database.System),
	} {
		session, err := pgx.Connect(t.Context(), dsn)
		require.NoError(t, err)

		var jit string
		err = session.QueryRow(t.Context(), "SHOW jit").Scan(&jit)
		_ = session.Close(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "off", jit, "a session that sends no jit parameter, as through PgBouncer")
	}
}

func currentSetting(ctx context.Context, conn *Connection, setting string) (string, error) {
	values, err := queryStrings(ctx, conn.Keel(ctx), "SELECT current_setting($1)", setting)
	if err != nil || len(values) == 0 {
		return "", err
	}

	return values[0], nil
}
