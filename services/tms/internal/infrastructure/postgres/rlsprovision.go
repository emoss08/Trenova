package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRLSProvisionRoleRequired = errors.New("a login role and password are required")
	ErrRLSProvisionSameRole     = errors.New(
		"the application and system roles must be different database roles",
	)
)

type RLSProvisionResult struct {
	KeyInstalled  bool
	PoliciesAdded int
}

func ProvisionRLS(
	ctx context.Context,
	db *keel.DB,
	cfg *config.Config,
) (*RLSProvisionResult, error) {
	result := new(RLSProvisionResult)
	if cfg == nil {
		return result, nil
	}

	added, err := keel.QueryValue[int](ctx, db, reconcileRLSSQL)
	if err != nil {
		return nil, fmt.Errorf("reconcile row-level security policies and grants: %w", err)
	}
	result.PoliciesAdded = added

	installed, err := InstallConfiguredRLSScopeKey(ctx, db, cfg.Database.RLS)
	if err != nil {
		return nil, err
	}
	result.KeyInstalled = installed

	return result, nil
}

const reconcileRLSSQL = "SELECT trenova_rls.reconcile()"

func InstallConfiguredRLSScopeKey(
	ctx context.Context,
	db *keel.DB,
	rls config.RLSConfig,
) (bool, error) {
	if !rls.Enabled() {
		return false, nil
	}

	key, err := rls.DecodeScopeKey()
	if err != nil {
		return false, err
	}

	if err = InstallRLSScopeKey(ctx, db, strings.TrimSpace(rls.ScopeKeyID), key); err != nil {
		return false, err
	}

	return true, nil
}

func InstallRLSScopeKey(ctx context.Context, db *keel.DB, keyID string, key []byte) error {
	if !scopeKeyIDPattern.MatchString(keyID) || len(key) < minScopeKeyBytes {
		return ErrInvalidScopeKey
	}

	pads := deriveScopePads(key)

	if _, err := keel.Exec(ctx, db, installScopeKeySQL, keyID, pads.Inner, pads.Outer); err != nil {
		return fmt.Errorf("install row-level security scope key %q: %w", keyID, err)
	}

	return nil
}

const installScopeKeySQL = `INSERT INTO trenova_rls.scope_keys (key_id, inner_pad, outer_pad)
VALUES ($1, $2, $3)
ON CONFLICT (key_id) DO UPDATE
SET inner_pad = EXCLUDED.inner_pad,
	outer_pad = EXCLUDED.outer_pad,
	retired_at = NULL
WHERE trenova_rls.scope_keys.inner_pad IS DISTINCT FROM EXCLUDED.inner_pad
	OR trenova_rls.scope_keys.outer_pad IS DISTINCT FROM EXCLUDED.outer_pad
	OR trenova_rls.scope_keys.retired_at IS NOT NULL`

func RetireRLSScopeKey(ctx context.Context, db *keel.DB, keyID string) (bool, error) {
	if !scopeKeyIDPattern.MatchString(keyID) {
		return false, ErrInvalidScopeKey
	}

	rows, err := keel.Exec(ctx, db, retireScopeKeySQL, keyID)
	if err != nil {
		return false, fmt.Errorf("retire row-level security scope key %q: %w", keyID, err)
	}

	return rows > 0, nil
}

const retireScopeKeySQL = `UPDATE trenova_rls.scope_keys
SET retired_at = extract(epoch from current_timestamp)::bigint
WHERE key_id = $1 AND retired_at IS NULL`

type RLSLoginRole struct {
	User     string
	Password string
}

type ProvisionRLSRolesParams struct {
	App    RLSLoginRole
	System RLSLoginRole
}

func ProvisionRLSRoles(
	ctx context.Context,
	db *keel.DB,
	params ProvisionRLSRolesParams,
) ([]string, error) {
	app := strings.TrimSpace(params.App.User)
	system := strings.TrimSpace(params.System.User)

	if app == "" || params.App.Password == "" || system == "" || params.System.Password == "" {
		return nil, ErrRLSProvisionRoleRequired
	}
	if strings.EqualFold(app, system) {
		return nil, ErrRLSProvisionSameRole
	}

	appVerifier, err := scramSHA256Verifier(params.App.Password)
	if err != nil {
		return nil, err
	}
	systemVerifier, err := scramSHA256Verifier(params.System.Password)
	if err != nil {
		return nil, err
	}

	roles := []struct {
		name     string
		verifier string
		member   string
		excluded string
	}{
		{name: app, verifier: appVerifier, member: rlsTenantRole, excluded: rlsOverrideRole},
		{name: system, verifier: systemVerifier, member: rlsOverrideRole, excluded: rlsTenantRole},
	}

	err = keel.Write(ctx, db, func(ctx context.Context) error {
		for _, role := range roles {
			if execErr := provisionLoginRole(ctx, db, role.name, role.verifier, role.member,
				role.excluded); execErr != nil {
				return fmt.Errorf("provision database role %q: %w", role.name, execErr)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return []string{app, system}, nil
}

const loginRoleAttributes = "LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS INHERIT"

func provisionLoginRole(ctx context.Context, db *keel.DB, name, verifier, member, excluded string) error {
	exists, err := keel.QueryValue[bool](ctx, db,
		"SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", name)
	if err != nil {
		return err
	}

	verb := "CREATE"
	if exists {
		verb = "ALTER"
	}
	if err = execFormatted(ctx, db,
		"SELECT format('%s ROLE %I %s PASSWORD %L', $1::text, $2::text, $3::text, $4::text)",
		verb, name, loginRoleAttributes, verifier); err != nil {
		return err
	}

	excludedMember, err := keel.QueryValue[bool](ctx, db,
		"SELECT pg_has_role($1, $2, 'MEMBER')", name, excluded)
	if err != nil {
		return err
	}
	if excludedMember {
		if err = execFormatted(ctx, db, "SELECT format('REVOKE %I FROM %I', $1::text, $2::text)",
			excluded, name); err != nil {
			return err
		}
	}

	isMember, err := keel.QueryValue[bool](ctx, db, "SELECT pg_has_role($1, $2, 'MEMBER')", name, member)
	if err != nil {
		return err
	}
	if isMember {
		return nil
	}

	return execFormatted(ctx, db, "SELECT format('GRANT %I TO %I', $1::text, $2::text)", member, name)
}

func execFormatted(ctx context.Context, db *keel.DB, format string, args ...any) error {
	statement, err := keel.QueryValue[string](ctx, db, format, args...)
	if err != nil {
		return err
	}

	_, err = keel.Exec(ctx, db, statement)
	return err
}

type RLSStatus struct {
	ActiveKeys        []string
	ProtectedTables   int
	GlobalTables      int
	UnprotectedTables []string
}

func ReadRLSStatus(ctx context.Context, db *keel.DB) (*RLSStatus, error) {
	status := new(RLSStatus)

	keys, err := keel.QueryAll(ctx, db, pgx.RowTo[string],
		`SELECT key_id FROM trenova_rls.scope_keys WHERE retired_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("read row-level security scope keys: %w", err)
	}
	status.ActiveKeys = keys

	if err = keel.QueryRows(ctx, db, func(row pgx.CollectableRow) error {
		return row.Scan(&status.ProtectedTables, &status.GlobalTables)
	}, `SELECT
			count(*) FILTER (WHERE c.relrowsecurity AND c.relforcerowsecurity),
			count(*) FILTER (WHERE g.table_name IS NOT NULL)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN trenova_rls.global_tables g ON g.table_name = c.relname
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p')`); err != nil {
		return nil, fmt.Errorf("count row-level security tables: %w", err)
	}

	unprotected, err := keel.QueryAll(ctx, db, pgx.RowTo[string],
		`SELECT c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
			AND c.relkind IN ('r', 'p')
			AND NOT (c.relrowsecurity AND c.relforcerowsecurity)
			AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
		ORDER BY c.relname`)
	if err != nil {
		return nil, fmt.Errorf("list tables without row-level security: %w", err)
	}
	status.UnprotectedTables = unprotected

	return status, nil
}
